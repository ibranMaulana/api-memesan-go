package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/memesan/api-react/internal/store"
)

func menu(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("passcode")
		switch requestValue(r, "act") {
		case "list":
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT mi.*,mc.tipe_harga,mc.taxPercentage FROM m_item mi JOIN m_company mc ON mc.passcode=mi.passcode WHERE mi.passcode=? AND mi.status_active='active' ORDER BY mi.item_code DESC", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "delete":
			id := r.URL.Query().Get("id")
			if !required(w, p, "passcode dibutuhkan") || !required(w, id, "id dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT id_item FROM m_item WHERE passcode=? AND id_item=?", p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 404, map[string]string{"msg": "menu tidak ditemukan"})
				return
			}
			_, e = s.DB.Exec("UPDATE m_item SET status_active='unactive' WHERE passcode=? AND id_item=?", p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]string{"status": "success"})
		case "set_in_stock":
			id := r.URL.Query().Get("id")
			if !required(w, p, "passcode dibutuhkan") || !required(w, id, "id dibutuhkan") {
				return
			}
			d := bodyMap(r)
			if d["in_stock"] == nil {
				writeJSON(w, 400, map[string]string{"msg": "in_stock dibutuhkan"})
				return
			}
			_, e := s.DB.Exec("UPDATE m_item SET in_stock=? WHERE passcode=? AND id_item=?", d["in_stock"], p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]string{"status": "success"})
		case "save", "update":
			act := requestValue(r, "act")
			var d map[string]any
			if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
				writeJSON(w, 400, map[string]string{"msg": "data dibutuhkan"})
				return
			}
			if d["passcode"] == nil {
				d["passcode"] = p
			}
			if d["item_name"] == nil || d["category_id"] == nil || d["item_code"] == nil {
				writeJSON(w, 400, map[string]string{"msg": "category_id dan item_name dibutuhkan"})
				return
			}
			id := d["id_item"]
			if act == "save" {
				var duplicate int
				_ = s.DB.QueryRow("SELECT COUNT(*) FROM m_item WHERE passcode=? AND item_code=? AND status_active='active'", d["passcode"], d["item_code"]).Scan(&duplicate)
				if duplicate > 0 {
					writeJSON(w, 404, map[string]string{"msg": "item code sudah ada"})
					return
				}
				id = newID("ITM")
				role := fmt.Sprint(d["print_role"])
				if role == "" {
					role = "KITCHEN"
				}
				if role == "CASHIER" {
					writeJSON(w, 400, map[string]string{"msg": "CASHIER bukan nilai yang valid untuk print_role"})
					return
				}
				_, e := s.DB.Exec("INSERT INTO m_item(id_item,passcode,category_id,item_name,item_code,net_price,ppn,price_after_ppn,discount_nominal,show_price_off,up_price,net_price_on,ppn_on,price_after_ppn_on,show_price,dpp,description,id_user,non_tax,hpp,hpp_on,print_role,has_sub_menu,sub_menu_group_id,status_active) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'active')", id, d["passcode"], d["category_id"], d["item_name"], d["item_code"], d["show_price_off"], d["ppn"], d["price_after_ppn"], d["discount_nominal"], d["show_price_off"], d["up_price"], d["show_price"], d["ppn_on"], d["price_after_ppn_on"], d["show_price"], d["dpp"], d["description"], d["id_user"], d["non_tax"], d["hpp"], d["hpp_on"], role, d["has_sub_menu"], d["sub_menu_group_id"])
				if e != nil {
					dbError(w, e)
					return
				}
			} else {
				var duplicate int
				_ = s.DB.QueryRow("SELECT COUNT(*) FROM m_item WHERE passcode=? AND item_code=? AND id_item<>? AND status_active='active'", d["passcode"], d["item_code"], id).Scan(&duplicate)
				if duplicate > 0 {
					writeJSON(w, 404, map[string]string{"msg": "item code sudah ada"})
					return
				}
				_, e := s.DB.Exec("UPDATE m_item SET category_id=?,item_name=?,item_code=?,net_price=?,net_price_on=?,show_price=?,show_price_off=?,up_price=?,hpp=?,hpp_on=?,non_tax=?,print_role=?,has_sub_menu=?,sub_menu_group_id=?,description=? WHERE id_item=? AND passcode=?", d["category_id"], d["item_name"], d["item_code"], d["show_price_off"], d["show_price"], d["show_price"], d["show_price_off"], d["up_price"], d["hpp"], d["hpp_on"], d["non_tax"], d["print_role"], d["has_sub_menu"], d["sub_menu_group_id"], d["description"], id, d["passcode"])
				if e != nil {
					dbError(w, e)
					return
				}
			}
			writeJSON(w, 200, map[string]any{"status": "success", "id_item": id})
		case "upload":
			id := r.URL.Query().Get("id")
			if id == "" {
				id = r.FormValue("id")
			}
			if !required(w, p, "passcode dibutuhkan") || !required(w, id, "id dibutuhkan") {
				return
			}
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				writeJSON(w, 400, map[string]string{"msg": "mohon upload file"})
				return
			}
			file, header, err := r.FormFile("foto")
			if err != nil {
				writeJSON(w, 400, map[string]string{"msg": "mohon upload file"})
				return
			}
			defer file.Close()
			if !strings.HasPrefix(header.Header.Get("Content-Type"), "image/jpeg") {
				writeJSON(w, 400, map[string]string{"msg": "extension file not allowed"})
				return
			}
			existing, err := s.Rows("SELECT id_item FROM m_item WHERE id_item=? AND passcode=?", id, p)
			if err != nil {
				dbError(w, err)
				return
			}
			if len(existing) == 0 {
				writeJSON(w, 404, map[string]string{"msg": "menu tidak ditemukan"})
				return
			}
			dir := filepath.Join("system", "assets", "images", "produk")
			if err = os.MkdirAll(dir, 0755); err != nil {
				dbError(w, err)
				return
			}
			name := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(header.Filename))
			dst, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				dbError(w, err)
				return
			}
			_, err = io.Copy(dst, file)
			closeErr := dst.Close()
			if err != nil {
				dbError(w, err)
				return
			}
			if closeErr != nil {
				dbError(w, closeErr)
				return
			}
			if _, err = s.DB.Exec("UPDATE m_item SET foto_1=? WHERE id_item=? AND passcode=?", name, id, p); err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, "success")
		default:
			invalidAction(w)
		}
	}
}

func reservasi(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act := requestValue(r, "act")
		p := r.URL.Query().Get("passcode")
		switch act {
		case "list":
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			page := ival(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			search := r.URL.Query().Get("search")
			q := "SELECT p.*,IFNULL(c.customer_name,'') customer_name FROM t_reservasi p LEFT JOIN m_customer c ON p.customer=c.customer_code WHERE p.passcode=?"
			args := []any{p}
			if search != "" {
				parts := strings.SplitN(search, " ", 2)
				if len(parts) == 2 {
					col := map[string]string{"c": "c.customer_name", "m": "p.meja", "s": "p.receipt_number"}[parts[0]]
					if col != "" {
						q += " AND " + col + " LIKE ?"
						args = append(args, "%"+parts[1]+"%")
					}
				}
			}
			q += " ORDER BY created_at DESC LIMIT 10 OFFSET ?"
			args = append(args, (page-1)*10)
			rows, e := s.Rows(q, args...)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "cancel":
			d := bodyMap(r)
			if !required(w, d["passcode"], "passcode dibutuhkan") || !required(w, d["id"], "id dibutuhkan") {
				return
			}
			_, e := s.DB.Exec("UPDATE t_reservasi SET status='cancel',cancel_by=? WHERE id_reservasi=? AND passcode=?", d["cancelBy"], d["id"], d["passcode"])
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
		case "cekid":
			rows, e := s.Rows("SELECT id_reservasi FROM t_reservasi WHERE passcode=? ORDER BY id_reservasi DESC LIMIT 1", p)
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 200, "")
			} else {
				writeJSON(w, 200, rows[0]["id_reservasi"])
			}
		case "save":
			var d map[string]any
			if json.NewDecoder(r.Body).Decode(&d) != nil {
				writeJSON(w, 400, "request tidak valid")
				return
			}
			p := fmt.Sprint(d["passcode"])
			if p == "" {
				writeJSON(w, 400, map[string]string{"msg": "passcode dibutuhkan"})
				return
			}
			items, _ := d["data"].([]any)
			if len(items) == 0 {
				writeJSON(w, 400, map[string]string{"msg": "data dibutuhkan"})
				return
			}
			ids, err := saveReservations(s, p, d, items)
			if err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"status": true, "trans": ids})
		default:
			invalidAction(w)
		}
	}
}

func saveReservations(s *store.Store, pass string, data map[string]any, items []any) ([]string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ids := []string{}
	for _, raw := range items {
		tr, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("transaksi tidak valid")
		}
		id := newID("RSV")
		created := fmt.Sprint(tr["date"]) + " " + fmt.Sprint(tr["time"])
		if created != " " {
			var old string
			if e := tx.QueryRow("SELECT id_reservasi FROM t_reservasi WHERE passcode=? AND created_at=? LIMIT 1", pass, created).Scan(&old); e == nil {
				ids = append(ids, old)
				continue
			}
		}
		number := id
		typ := fmt.Sprint(tr["transactionType"])
		detailRows, _ := tr["item"].([]any)
		subtotal, dpp, nonTax, service := 0.0, 0.0, 0.0, 0.0
		for _, iv := range detailRows {
			it, ok := iv.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("item reservasi tidak valid")
			}
			qty := numValue(it["qty"])
			if qty == 0 {
				qty = 1
			}
			price := numValue(it["price"])
			if typ == "Online" && numValue(it["net_price_on"]) > 0 {
				price = numValue(it["net_price_on"])
			}
			svc := numValue(it["service"])
			if typ == "Offline" {
				svc = 0
			}
			line := price * qty
			subtotal += line
			service += svc * qty
			if boolValue(it["nonTax"]) {
				nonTax += line
			} else {
				dpp += line
			}
			if _, err = tx.Exec("INSERT INTO t_reservasi_detail(id_reservasi,nomor_transaksi,catatan,id_item,qty,net_price,ppn,price_after_ppn,discount_nominal,up_price,net_price_on,ppn_on,price_after_ppn_on,show_price,show_price_off,dpp,total,passcode,type_transaction,non_tax) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, number, it["catatan"], it["id"], qty, it["price"], numValue(it["priceWithPPN"])-numValue(it["price"]), it["priceWithPPN"], it["discount"], svc, it["net_price_on"], it["ppn_on"], it["price_after_ppn_on"], it["show_price"], it["show_price_off"], line, line, pass, typ, boolValue(it["nonTax"])); err != nil {
				return nil, err
			}
		}
		status := fmt.Sprint(tr["status"])
		if fmt.Sprint(tr["cancelBy"]) == "" {
			status = "success"
		}
		customer := ""
		if c, ok := tr["customer"].(map[string]any); ok {
			customer = fmt.Sprint(c["id"])
		}
		operator, user := "", ""
		if o, ok := tr["operator"].(map[string]any); ok {
			operator = fmt.Sprint(o["fullname"])
			user = fmt.Sprint(o["id"])
		}
		ppn := dpp * 0.1
		_, err = tx.Exec("INSERT INTO t_reservasi(id_reservasi,nomor_transaksi,receipt_number,id_branch,passcode,meja,tgl_reservasi,customer,sales,user,payment_method,subtotal,discount,total_discount,ppn,total,donation,payment_amount,created_at,status,cancel_by,type_transaction,tipe_harga,total_upprice,total_net,total_ppn,total_non_tax,grand_total,dine_in,nama_tamu,service) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, number, tr["receiptNumber"], data["id_branch"], pass, tr["table"], tr["date"], customer, operator, user, tr["paymentMethode"], tr["subTotal"], tr["discount"], tr["totalDiscount"], tr["PPN"], numValue(tr["subTotal"])-numValue(tr["discount"]), tr["donation"], tr["paymentAmount"], created, status, tr["cancelBy"], typ, firstValue(data["tipe_harga"]), service, dpp, ppn, nonTax, tr["total"], tr["dineIn"], tr["namaTamu"], data["with_service"])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}
func numValue(v any) float64 { f, _ := strconv.ParseFloat(fmt.Sprint(v), 64); return f }
func boolValue(v any) bool   { return v == true || fmt.Sprint(v) == "1" || fmt.Sprint(v) == "true" }
func firstValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
