package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/memesan/api-react/internal/store"
)

// rekap deliberately uses one calculation for both endpoints.  The update
// endpoint only changes persistence; its response remains the same settlement
// document as mob_rekap.php.
func rekap(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil && r.URL.Path != "/mob_rekap.php" {
			writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
			return
		}
		p := rekapValue(in["passcode"], r.URL.Query().Get("passcode"))
		emp := rekapValue(in["employees_id"], r.URL.Query().Get("employees_id"))
		if p == "" {
			writeJSON(w, 400, map[string]string{"msg": "passcode dibutuhkan"})
			return
		}
		if emp == "" {
			writeJSON(w, 400, map[string]string{"msg": "employees_id dibutuhkan"})
			return
		}
		if r.URL.Query().Get("act") == "" && rekapValue(in["aktual_akhir"], "") == "" {
			writeJSON(w, 400, map[string]string{"msg": "nilai kas aktual akhir dibutuhkan"})
			return
		}
		result, err := buildRekap(s, p, emp, in)
		if err != nil {
			dbError(w, err)
			return
		}
		if err = persistRekap(s, strings.HasSuffix(r.URL.Path, "mob_rekap_update.php") || strings.HasSuffix(r.URL.Path, "mob_rekap_update"), p, emp, in, result); err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, result)
	}
}

func buildRekap(s *store.Store, pass, emp string, in map[string]any) (map[string]any, error) {
	start, end := time.Now().Format("2006-01-02 00:00:00"), time.Now().Format("2006-01-02 23:59:59")
	if v := rekapValue(in["tanggal_awal"], ""); v != "" {
		start = mysqlDate(v) + " 00:00:00"
	}
	if v := rekapValue(in["tanggal_akhir"], ""); v != "" {
		end = mysqlDate(v) + " 23:59:59"
	}
	if id := rekapValue(in["id_rekap"], ""); id != "" {
		var e error
		var a, b string
		e = s.DB.QueryRow("SELECT jam_buka,jam_tutup FROM r_rekap WHERE id_rekap=? AND passcode=?", id, pass).Scan(&a, &b)
		if e != nil {
			return nil, fmt.Errorf("id rekap tidak ditemukan")
		}
		start, end = a, b
	}
	rows, e := s.Rows(`SELECT p.type_transaction,p.status,p.payment_method,p.grand_total,p.discount,p.total_upprice,p.total_net,p.total_ppn,p.total_non_tax,p.donation,p.dine_in,p.total, d.id_item,d.qty,d.total detail_total,d.ppn,d.dpp,d.hpp,d.hpp_on,d.discount_nominal,i.item_name,i.item_code,i.category_id,c.category FROM t_penjualan p LEFT JOIN t_penjualan_detail d ON d.id_penjualan=p.id_penjualan LEFT JOIN m_item i ON i.id_item=d.id_item LEFT JOIN m_category c ON c.id=i.category_id WHERE p.passcode=? AND p.created_at BETWEEN ? AND ?`, pass, start, end)
	if e != nil {
		return nil, e
	}
	cat := map[string]map[string]any{}
	seenSales := map[string]bool{}
	off, on := []map[string]any{}, []map[string]any{}
	var total, cash, discount, service, ppn, nonTax, dpp, hpp, hppOn, cancel, cancelQty, dine, take float64
	for _, x := range rows {
		status := strings.ToLower(rekapString(x["status"]))
		gt := nf(x["grand_total"])
		if status == "cancel" {
			cancel += gt
			cancelQty++
			continue
		}
		saleID := rekapString(x["id_penjualan"])
		if !seenSales[saleID] {
			seenSales[saleID] = true
			total += gt
			discount += nf(x["discount"])
			service += nf(x["total_upprice"])
			ppn += nf(x["total_ppn"])
			nonTax += nf(x["total_non_tax"])
			if strings.EqualFold(rekapString(x["payment_method"]), "tunai") {
				cash += gt
			}
			if nf(x["dine_in"]) == 1 {
				dine++
			} else {
				take++
			}
		}
		if x["id_item"] != nil {
			item := map[string]any{"id_item": x["id_item"], "item_name": x["item_name"], "item_code": x["item_code"], "jml": nf(x["qty"]), "total": nf(x["detail_total"]), "dpp": nf(x["dpp"]), "ppn": nf(x["ppn"]), "hpp": nf(x["hpp"]), "hpp_on": nf(x["hpp_on"]), "discount_nominal": nf(x["discount_nominal"])}
			if rekapString(x["type_transaction"]) == "Onl" {
				on = append(on, item)
				hppOn += nf(x["hpp_on"])
			} else {
				off = append(off, item)
				hpp += nf(x["hpp"])
			}
			dpp += nf(x["dpp"])
			key := rekapString(x["category_id"])
			if key != "" {
				if cat[key] == nil {
					cat[key] = map[string]any{"id": x["category_id"], "category": x["category"], "jml": 0.0, "total": 0.0}
				}
				cat[key]["jml"] = nf(cat[key]["jml"]) + nf(x["qty"])
				cat[key]["total"] = nf(cat[key]["total"]) + nf(x["detail_total"])
			}
		}
	}
	kasIn, kasOut := []map[string]any{}, []map[string]any{}
	var kin, kout float64
	kr, e := s.Rows("SELECT id_kas,nominal,description,tipe FROM a_kas WHERE passcode=? AND log_time BETWEEN ? AND ?", pass, start, end)
	if e != nil {
		return nil, e
	}
	for _, x := range kr {
		if strings.EqualFold(rekapString(x["tipe"]), "IN") {
			kasIn = append(kasIn, x)
			kin += nf(x["nominal"])
		} else {
			kasOut = append(kasOut, x)
			kout += nf(x["nominal"])
		}
	}
	refund := 0.0
	_ = s.DB.QueryRow("SELECT COALESCE(SUM(r.total_refund),0) FROM refund_transaction r JOIN t_penjualan p ON p.id_penjualan=r.original_transaction_id WHERE p.passcode=? AND p.created_at BETWEEN ? AND ?", pass, start, end).Scan(&refund)
	net := total - refund
	saldoTunai := cash + kin - kout
	catRows := make([]map[string]any, 0, len(cat))
	for _, v := range cat {
		catRows = append(catRows, v)
	}
	return map[string]any{"penjualan": map[string]any{"categories": catRows, "cancel": map[string]any{"qty": cancelQty, "total": cancel}, "offline": map[string]any{"items": off, "subtotal": sumItems(off), "diskon": discount, "non_tax": nonTax, "hpp": hpp, "dpp": dpp, "pajak": ppn, "total": total, "profit": dpp - hpp}, "online": map[string]any{"items": on, "subtotal": sumItems(on), "diskon": 0, "non_tax": 0, "hpp": hppOn, "dpp": dpp, "pajak": ppn, "total": total, "profit": dpp - hppOn}, "diskon": discount, "service": service, "non_tax": nonTax, "total_dpp": dpp, "pajak": ppn, "donation": 0, "total": net, "total_dine_in": dine, "total_take_away": take, "total_hpp": hpp + hppOn, "total_profit": dpp - hpp - hppOn}, "kas_in": map[string]any{"items": kasIn, "total": kin}, "kas_out": map[string]any{"items": kasOut, "total": kout}, "employees_id": emp, "passcode": pass, "saldo_awal": 0, "saldo_akhir_tunai": saldoTunai, "saldo_akhir_nontunai": net - cash, "aktual_awal": 0, "aktual_akhir": nf(in["aktual_akhir"]), "jam_buka": start, "jam_tutup": end, "totalRefund": refund, "totalBersih": net, "subtotal": sumItems(off) + sumItems(on), "diskon": discount, "non_tax": nonTax, "dpp": dpp, "ppn": ppn, "grand_total": net, "selisih": nf(in["aktual_akhir"]) - saldoTunai}, nil
}

func persistRekap(s *store.Store, update bool, pass, emp string, in, result map[string]any) error {
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if ks, ok := in["kas"].([]any); ok && !update {
		for _, v := range ks {
			x, _ := v.(map[string]any)
			if _, e = tx.Exec("INSERT INTO a_kas(nominal,description,tipe,id_branch,passcode,employees_id,log_time) VALUES(?,?,?,'Pusat',?,?,?)", x["nominal"], x["description"], x["tipe"], pass, emp, x["log_time"]); e != nil {
				return e
			}
		}
	}
	b, _ := json.Marshal(result)
	id := rekapValue(in["id_rekap"], "")
	if update {
		if id == "" {
			return fmt.Errorf("id_rekap dibutuhkan")
		}
		_, e = tx.Exec("UPDATE r_rekap SET employees_id=?,aktual_akhir=?,selisih=?,grand_total=?,subtotal=?,diskon=?,dpp=?,ppn=?,data=? WHERE id_rekap=? AND passcode=?", emp, result["aktual_akhir"], result["selisih"], result["grand_total"], result["subtotal"], result["diskon"], result["dpp"], result["ppn"], b, id, pass)
	} else {
		id = newID("RKP")
		_, e = tx.Exec("INSERT INTO r_rekap(id_rekap,passcode,employees_id,saldo_awal,subtotal,diskon,non_tax,dpp,ppn,selisih,grand_total,saldo_akhir_nontunai,saldo_akhir_tunai,aktual_awal,aktual_akhir,jam_buka,jam_tutup,data,multidate) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, pass, emp, result["saldo_awal"], result["subtotal"], result["diskon"], result["non_tax"], result["dpp"], result["ppn"], result["selisih"], result["grand_total"], result["saldo_akhir_nontunai"], result["saldo_akhir_tunai"], result["aktual_awal"], result["aktual_akhir"], result["jam_buka"], result["jam_tutup"], b, false)
		result["id_rekap"] = id
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}

func rekapValue(v any, fallback string) string {
	if s := fmt.Sprint(v); v != nil && s != "<nil>" && s != "" {
		return s
	}
	return fallback
}
func nf(v any) float64 { f, _ := strconv.ParseFloat(fmt.Sprint(v), 64); return f }
func sumItems(a []map[string]any) float64 {
	var n float64
	for _, x := range a {
		n += nf(x["total"])
	}
	return n
}
func rekapString(v any) string { return rekapValue(v, "") }
