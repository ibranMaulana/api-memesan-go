package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/memesan/api-react/internal/store"
)

func hutang(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		switch act {
		case "hutangCabang":
			rows, e := s.Rows("SELECT SUM(p.total_after_pajak)-SUM(p.dp+p.paid_off) hutang,b.id_branch,b.branch_name FROM t_penjualan p JOIN m_branch b ON b.id_branch=p.id_branch WHERE p.total_after_pajak != p.dp+p.paid_off AND p.passcode=? GROUP BY p.id_branch", p)
			if e != nil {
				dbError(w, e)
				return
			}
			returnHutang(w, rows, "id_branch", "branch_name")
		case "hutangCustomer":
			rows, e := s.Rows("SELECT SUM(p.total_after_pajak)-SUM(p.dp+p.paid_off) hutang,c.customer_code,c.customer_name FROM t_penjualan p JOIN m_customer c ON c.customer_code=p.customer WHERE p.total_after_pajak != p.dp+p.paid_off AND p.passcode=? AND p.id_branch=? GROUP BY p.customer_code,c.customer_name", p, r.URL.Query().Get("id_branch"))
			if e != nil {
				dbError(w, e)
				return
			}
			returnHutang(w, rows, "customer_code", "customer_name")
		case "detailHutangCustomer":
			rows, e := s.Rows("SELECT SUM(total_after_pajak)-SUM(dp+paid_off) hutang,tgl_penjualan,id_penjualan,nomor_transaksi FROM t_penjualan WHERE total_after_pajak != dp+paid_off AND passcode=? AND id_branch=? AND customer=? GROUP BY id_penjualan,nomor_transaksi,tgl_penjualan", p, r.URL.Query().Get("id_branch"), r.URL.Query().Get("customer_code"))
			if e != nil {
				dbError(w, e)
				return
			}
			returnHutang(w, rows, "id_penjualan", "nomor_transaksi")
		case "historyDetailHutangCustomer":
			rows, e := s.Rows("SELECT DATE_FORMAT(tanggal,'%d/%m') tgl_transaksi,0 credit,dibayar debit FROM a_receive_payment WHERE id_transaksi=?", r.URL.Query().Get("nomor_transaksi"))
			if e != nil {
				dbError(w, e)
				return
			}
			base, e := s.Rows("SELECT DATE_FORMAT(tgl_penjualan,'%d/%m') tgl_transaksi,total_after_pajak credit,dp debit FROM t_penjualan WHERE nomor_transaksi=?", r.URL.Query().Get("nomor_transaksi"))
			if e != nil {
				dbError(w, e)
				return
			}
			out := make([]map[string]any, 0, len(base)+len(rows)+1)
			var credit, debit float64
			for _, x := range base {
				out = append(out, x)
				credit += number(x["credit"])
				debit += number(x["debit"])
			}
			for _, x := range rows {
				out = append(out, x)
				debit += number(x["debit"])
			}
			out = append(out, map[string]any{"tgl_penjualan": "Sisa Hutang", "credit": credit - debit, "debit": 0})
			writeJSON(w, 200, map[string]any{"data": out})
		case "receivePayment":
			if err := r.ParseForm(); err != nil {
				writeJSON(w, 400, "request tidak valid")
				return
			}
			paid := r.FormValue("dibayar")
			if paid == "" {
				writeJSON(w, 200, "Gagal : Nominal belum dimasukkan!")
				return
			}
			var branch, code, customer, saleDate, dueDate, pass string
			e := s.DB.QueryRow("SELECT p.id_branch,b.branch_code,p.customer,p.tgl_penjualan,p.tgl_jatuh_tempo,p.passcode FROM t_penjualan p JOIN m_branch b ON b.id_branch=p.id_branch WHERE p.nomor_transaksi=?", r.FormValue("nomor_transaksi")).Scan(&branch, &code, &customer, &saleDate, &dueDate, &pass)
			if e != nil {
				dbError(w, e)
				return
			}
			id := newID("RCP")
			tx, e := s.DB.Begin()
			if e != nil {
				dbError(w, e)
				return
			}
			defer tx.Rollback()
			_, e = tx.Exec("INSERT INTO a_receive_payment(id_receive_payment,id_transaksi,id_branch,customer_code,tanggal,tgl_penjualan,tgl_jatuh_tempo,keterangan,payment_method,dibayar,user,passcode) VALUES(?,?,?, ?,CURRENT_DATE(),?,?, 'Mob','Tunai',?,?,?)", id, r.FormValue("nomor_transaksi"), branch, customer, saleDate, dueDate, paid, r.FormValue("employees_id"), pass)
			if e != nil {
				dbError(w, e)
				return
			}
			_, e = tx.Exec("UPDATE t_penjualan SET paid_off=(SELECT COALESCE(SUM(dibayar),0) FROM a_receive_payment WHERE id_transaksi=?) WHERE nomor_transaksi=?", r.FormValue("nomor_transaksi"), r.FormValue("nomor_transaksi"))
			if e != nil {
				dbError(w, e)
				return
			}
			if e = tx.Commit(); e != nil {
				dbError(w, e)
				return
			}
			_ = code
			writeJSON(w, 200, id)
		default:
			invalidAction(w)
		}
	}
}

func returnHutang(w http.ResponseWriter, rows []map[string]any, idKey, nameKey string) {
	var total float64
	for _, x := range rows {
		total += number(x["hutang"])
	}
	rows = append(rows, map[string]any{idKey: "All", nameKey: "Semua", "hutang": total})
	writeJSON(w, 200, map[string]any{"data": rows})
}
func number(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case []byte:
		f, _ := strconv.ParseFloat(string(x), 64)
		return f
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	}
	return 0
}

func kartuStock(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		if act == "dataCabang" {
			rows, e := s.Rows("SELECT id_branch,branch_name FROM m_branch WHERE passcode=?", p)
			if e != nil {
				dbError(w, e)
				return
			}
			for _, x := range rows {
				var v float64
				e = s.DB.QueryRow("SELECT COALESCE(SUM(qty_in*price)-SUM(qty_out*price),0) FROM t_stock_movement WHERE passcode=? AND id_branch=?", p, x["id_branch"]).Scan(&v)
				if e != nil {
					dbError(w, e)
					return
				}
				x["branch_name"] = fmt.Sprintf("%s (Rp. %s)", x["branch_name"], formatIDR(v))
			}
			writeJSON(w, 200, map[string]any{"data": rows})
			return
		}
		if act != "kartuStock" {
			invalidAction(w)
			return
		}
		branch, item := r.URL.Query().Get("id_branch"), r.URL.Query().Get("id_item")
		t1, t2 := mysqlDate(r.URL.Query().Get("t1")), mysqlDate(r.URL.Query().Get("t2"))
		filter, args := "", []any{p, item, t1}
		if branch != "" && branch != "semua" {
			filter = " AND id_branch=?"
			args = append(args, branch)
		}
		var in0, out0 float64
		_ = s.DB.QueryRow("SELECT COALESCE(SUM(qty_in),0),COALESCE(SUM(qty_out),0) FROM t_stock_movement WHERE passcode=? AND id_item=? AND date<?"+filter, args...).Scan(&in0, &out0)
		stock := in0 - out0
		before := []map[string]any{{"id_transaction": "Before", "qty_in": in0, "qty_out": out0, "date": "", "stock": stock}}
		qargs := []any{p, t1, t2, item}
		q := "SELECT s.id_transaction,i.item_name,DATE_FORMAT(s.date,'%d/%m/%Y') date,s.qty_in,s.qty_out FROM t_stock_movement s JOIN m_item i ON i.id_item=s.id_item WHERE s.passcode=? AND s.date>=? AND s.date<=? AND s.id_item=?"
		if branch != "" && branch != "semua" {
			q += " AND s.id_branch=?"
			qargs = append(qargs, branch)
		}
		q += " ORDER BY s.date,s.id_transaction"
		rows, e := s.Rows(q, qargs...)
		if e != nil {
			dbError(w, e)
			return
		}
		for _, x := range rows {
			stock += number(x["qty_in"]) - number(x["qty_out"])
			x["stock"] = stock
		}
		writeJSON(w, 200, map[string]any{"before": before, "now": rows, "after": []map[string]any{}, "stock": []map[string]any{{"id_transaction": "Stock", "qty_in": in0, "qty_out": out0, "date": "", "stock": stock}}})
	}
}

func mysqlDate(v string) string {
	p := strings.Split(v, "/")
	if len(p) == 3 {
		return p[2] + "-" + p[1] + "-" + p[0]
	}
	return v
}
func formatIDR(v float64) string { return fmt.Sprintf("%.0f", v) }

func priceList(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		if act == "test" {
			writeJSON(w, 200, "success")
			return
		}
		if act == "category" {
			rows, e := s.Rows("SELECT * FROM m_category WHERE passcode=? AND status_active='active'", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
			return
		}
		if act == "uploadFotoProduct" {
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
				return
			}
			id := r.FormValue("xid")
			if id == "" {
				id = r.URL.Query().Get("xid")
			}
			if id == "" {
				writeJSON(w, 400, map[string]string{"msg": "xid dibutuhkan"})
				return
			}
			if _, err := s.Rows("SELECT id_item FROM m_item WHERE id_item=? AND passcode=?", id, p); err != nil {
				dbError(w, err)
				return
			}
			dir := "system/assets/images/produk"
			if err := os.MkdirAll(dir, 0755); err != nil {
				dbError(w, err)
				return
			}
			for i := 1; i <= 4; i++ {
				raw := r.FormValue(fmt.Sprintf("foto_%d", i))
				if raw == "" {
					continue
				}
				b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, "data:image/jpg;base64,"))
				if err != nil {
					writeJSON(w, 400, map[string]string{"msg": "foto tidak valid"})
					return
				}
				name := strings.ReplaceAll(id, "/", "") + fmt.Sprintf("_%d.jpeg", i)
				if err = os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
					dbError(w, err)
					return
				}
				if _, err = s.DB.Exec(fmt.Sprintf("UPDATE m_item SET foto_%d=? WHERE id_item=? AND passcode=?", i), name, id, p); err != nil {
					dbError(w, err)
					return
				}
			}
			writeJSON(w, 200, "success")
			return
		}
		if act != "getProduct" && act != "searchProduct" {
			invalidAction(w)
			return
		}
		page := ival(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		where := []string{"i.passcode=?", "i.status_active='active'"}
		args := []any{p}
		if act == "searchProduct" {
			name := r.URL.Query().Get("name")
			where = append(where, "(i.item_name LIKE ? OR i.item_code LIKE ?)")
			args = append(args, "%"+name+"%", "%"+name+"%")
			if c := r.URL.Query().Get("category"); c != "" {
				where = append(where, "i.category_name LIKE ?")
				args = append(args, "%"+c+"%")
			}
		}
		q := "SELECT i.id_item id,i.item_code,i.item_name,i.merk,i.uom,i.specification,i.net_price sell_price,i.base_price,i.net_price wholesale,i.sell_price msrp,COALESCE(s.stock,0) stock,i.foto_1,i.foto_2,i.foto_3,i.foto_4 FROM m_item i LEFT JOIN (SELECT id_item,SUM(qty_in)-SUM(qty_out) stock FROM t_stock_movement WHERE passcode=? GROUP BY id_item) s ON s.id_item=i.id_item WHERE " + strings.Join(where, " AND ") + " ORDER BY i.item_name LIMIT 9 OFFSET ?"
		args = append([]any{p}, args...)
		args = append(args, (page-1)*9)
		rows, e := s.Rows(q, args...)
		if e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"data": rows, "page": page})
	}
}

func pembelian(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		if act == "getSupplier" {
			rows, e := s.Rows("SELECT id_supplier,supplier_code,supplier_name FROM m_supplier WHERE passcode=? AND status_active='active' ORDER BY supplier_name", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
			return
		}
		if act != "submit" {
			invalidAction(w)
			return
		}
		if e := r.ParseForm(); e != nil {
			writeJSON(w, 400, "request tidak valid")
			return
		}
		p = r.FormValue("passcode")
		var items []struct {
			ID    any     `json:"id"`
			Qty   float64 `json:"qty"`
			Price float64 `json:"price"`
		}
		if json.Unmarshal([]byte(r.FormValue("items")), &items) != nil || len(items) == 0 {
			writeJSON(w, 400, "items diperlukan")
			return
		}
		total, _ := strconv.ParseFloat(r.FormValue("total"), 64)
		discount, _ := strconv.ParseFloat(r.FormValue("diskon"), 64)
		tax, _ := strconv.ParseFloat(r.FormValue("pajak"), 64)
		nd := total * discount / 100
		after := total - nd
		nt := after * tax / 100
		grand := after + nt
		po := newID("PMB")
		sys := newID("PMB")
		tx, e := s.DB.Begin()
		if e != nil {
			dbError(w, e)
			return
		}
		defer tx.Rollback()
		for _, i := range items {
			_, e = tx.Exec("INSERT INTO t_purchase_order_detail(id_purchase_order,id_system_purchase_order,id_item,qty,base_price,total,passcode) VALUES(?,?,?,?,?,?,?)", po, sys, i.ID, i.Qty, i.Price, i.Qty*i.Price, p)
			if e != nil {
				dbError(w, e)
				return
			}
		}
		_, e = tx.Exec("INSERT INTO t_purchase_order(id_purchase_order,id_system_purchase_order,tgl_po,tgl_jatuh_tempo,id_supplier,employees_id,status_pembayaran,tipe_pembayaran,grand_total,discount,nominal_discount,total_after_discount,pajak,nominal_pajak,total_after_pajak,paid_off,note,passcode,status_penerimaan,id_branch) VALUES(?,?,CURRENT_DATE(),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", po, sys, r.FormValue("jatuh_tempo"), r.FormValue("id_supplier"), r.FormValue("admin"), r.FormValue("status_pembayaran"), r.FormValue("tipe_pembayaran"), grand, discount, nd, after, tax, nt, grand, grand, r.FormValue("catatan"), p, "Complete", r.FormValue("id_branch"))
		if e != nil {
			dbError(w, e)
			return
		}
		if e = tx.Commit(); e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, "success")
	}
}

func invalidAction(w http.ResponseWriter) {
	writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
}
