package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/memesan/api-react/internal/store"
)

func product(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		if act == "test" {
			writeJSON(w, 200, "success")
			return
		}
		switch act {
		case "category":
			rows, e := s.Rows("SELECT * FROM m_category WHERE passcode=?", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "productDetail":
			id := r.URL.Query().Get("id_item")
			rows, e := s.Rows("SELECT foto_1,foto_2,foto_3,foto_4,item_code,category_name,item_name,base_price hpp,net_price ws,sell_price mrsp,merk,uom satuan,weight,description FROM m_item WHERE id_item=? AND passcode=?", id, p)
			if e != nil {
				dbError(w, e)
				return
			}
			promos, _ := s.Rows("SELECT p.id_promo,p.nama_promo FROM m_promo p JOIN m_promo_detail pd ON p.id_promo=pd.id_promo WHERE pd.id_item=? AND p.passcode=?", id, p)
			stock, _ := s.Rows("SELECT COALESCE(SUM(qty_in-qty_out),0) stock FROM t_stock_movement WHERE id_item=? AND passcode=?", id, p)
			writeJSON(w, 200, map[string]any{"data_product": rows, "promo": promos, "stock": stock})
		case "searchProduct":
			name := r.URL.Query().Get("name")
			rows, e := s.Rows("SELECT id_item id,item_code,item_name,merk,specification,net_price sell_price,foto_1,foto_2,foto_3,foto_4 FROM m_item WHERE passcode=? AND status_active='active' AND (item_name LIKE ? OR item_code LIKE ?) ORDER BY item_name", p, "%"+name+"%", "%"+name+"%")
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows})
		case "getProductReady":
			rows, e := s.Rows("SELECT id_item id,item_code,item_name,merk,sell_price mrsp,foto_1 FROM m_item WHERE passcode=? ORDER BY item_name", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows, "page": 1})
		case "productUpdate":
			d := bodyMap(r)
			id := d["id_item"]
			if id == nil {
				id = r.URL.Query().Get("id_item")
			}
			_, e := s.DB.Exec("UPDATE m_item SET item_name=?,item_code=?,category_id=?,net_price=?,net_price_on=?,description=? WHERE id_item=? AND passcode=?", d["item_name"], d["item_code"], d["category_id"], d["net_price"], d["net_price_on"], d["description"], id, p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
		case "productSave", "uploadFotoProduct":
			if err := r.ParseForm(); err != nil {
				writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
				return
			}
			if act == "productSave" {
				var duplicate int
				_ = s.DB.QueryRow("SELECT COUNT(*) FROM m_item WHERE item_code=? AND passcode=?", r.FormValue("item_code"), p).Scan(&duplicate)
				if duplicate > 0 {
					writeJSON(w, 200, "Maaf kode produk sudah digunakan")
					return
				}
				id := newID("ITM")
				_, err := s.DB.Exec("INSERT INTO m_item(id_item,item_code,item_name,category_name,merk,uom,weight,base_price,net_price,net_price_on,sell_price,description,passcode,status_active) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,'active')", id, r.FormValue("item_code"), r.FormValue("item_name"), r.FormValue("category_name"), r.FormValue("merk"), r.FormValue("uom"), r.FormValue("weight"), r.FormValue("base_price"), r.FormValue("net_price"), r.FormValue("net_price_on"), r.FormValue("sell_price"), r.FormValue("description"), p)
				if err != nil {
					dbError(w, err)
					return
				}
				if err := saveProductPhotos(s, p, id, r.Form, r.FormValue("item_name")); err != nil {
					dbError(w, err)
					return
				}
				writeJSON(w, 200, "success")
				return
			}
			id := r.FormValue("xid")
			if id == "" {
				id = r.FormValue("id_item")
			}
			if id == "" {
				writeJSON(w, 400, map[string]string{"msg": "id_item dibutuhkan"})
				return
			}
			if err := saveProductPhotos(s, p, id, r.Form, r.FormValue("item_name")); err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, "success")
		default:
			invalidAction(w)
		}
	}
}

func saveProductPhotos(s *store.Store, passcode, id string, values map[string][]string, name string) error {
	var oldID string
	if err := s.DB.QueryRow("SELECT id_item FROM m_item WHERE id_item=? AND passcode=?", id, passcode).Scan(&oldID); err != nil {
		return err
	}
	dir := filepath.Join("system", "assets", "images", "produk")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	clean := strings.ReplaceAll(name, " ", "")
	if len(clean) > 5 {
		clean = clean[:5]
	}
	for i := 1; i <= 4; i++ {
		raw := values[fmt.Sprintf("foto_%d", i)]
		if len(raw) == 0 || raw[0] == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw[0], "data:image/jpg;base64,"))
		if err != nil {
			return err
		}
		filename := strings.ReplaceAll(id, "/", "") + "_" + clean + "_" + fmt.Sprint(i) + ".jpg"
		if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
			return err
		}
		if _, err := s.DB.Exec(fmt.Sprintf("UPDATE m_item SET foto_%d=? WHERE id_item=? AND passcode=?", i), filename, id, passcode); err != nil {
			return err
		}
	}
	return nil
}

func dapur(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p, d := requestValue(r, "act"), r.URL.Query().Get("passcode"), r.URL.Query().Get("dapur")
		if act == "ready" || act == "ready_bulk" || act == "reject" || act == "reject_bulk" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
				return
			}
			status := "ready"
			stockType := "PEN"
			if strings.HasPrefix(act, "reject") {
				status = "reject"
				stockType = "REJ"
			}
			ids := []string{}
			if act == "ready_bulk" || act == "reject_bulk" {
				if list, ok := body["data"].([]any); ok {
					for _, v := range list {
						x, _ := v.(map[string]any)
						ids = append(ids, fmt.Sprint(x["id_item"]))
					}
				}
			} else if list, ok := body["id_items"].([]any); ok {
				for _, v := range list {
					ids = append(ids, fmt.Sprint(v))
				}
			}
			if len(ids) == 0 {
				writeJSON(w, 400, map[string]string{"msg": "id_items dibutuhkan"})
				return
			}
			if err := updateDapurItems(s, p, body, status, stockType, ids); err != nil {
				dbError(w, err)
				return
			}
			writeJSON(w, 200, true)
			return
		}
		if act == "list" || act == "list_by_items" || act == "list_rekap" {
			q := "SELECT p.id_penjualan,p.nomor_transaksi,p.receipt_number,p.id_branch,p.passcode,p.keterangan,p.meja,p.tgl_penjualan,p.customer,IFNULL(c.customer_name,'') customer_name,p.sales,p.user,p.grand_total total,p.payment_amount,p.status,p.created_at FROM t_penjualan p LEFT JOIN m_customer c ON p.customer=c.customer_code WHERE p.passcode=? AND p.status='success' AND p.tgl_penjualan=CURDATE() AND EXISTS (SELECT 1 FROM t_penjualan_detail d JOIN m_item i ON i.id_item=d.id_item JOIN m_category mc ON mc.id=i.category_id WHERE d.id_penjualan=p.id_penjualan AND mc.print_role=? AND d.status_dapur!='ready'"
			args := []any{p, d}
			if act == "list" {
				q += " AND d.status_dapur='new'"
			}
			if act == "list_by_items" {
				q += " AND d.id_item=?"
				args = append(args, r.URL.Query().Get("item"))
			}
			q += ") ORDER BY p.nomor_transaksi ASC LIMIT 5"
			headers, e := s.Rows(q, args...)
			if e != nil {
				dbError(w, e)
				return
			}
			for _, h := range headers {
				details, _ := s.Rows("SELECT d.id_penjualan_detail,d.id_penjualan,d.nomor_transaksi,d.catatan,d.id_item,d.qty,d.total,mi.item_name,mi.item_code,t.meja,t.dine_in,d.status_dapur,d.ready_at,d.reject_at,DATE_FORMAT(d.ready_at,'%H:%i:%s') jam_ready,DATE_FORMAT(d.reject_at,'%H:%i:%s') jam_reject FROM t_penjualan_detail d JOIN m_item mi ON mi.id_item=d.id_item JOIN t_penjualan t ON t.id_penjualan=d.id_penjualan JOIN m_category mc ON mc.id=mi.category_id WHERE d.id_penjualan=? AND mc.print_role=? AND d.status_dapur!='ready' ORDER BY mi.item_code", h["id_penjualan"], d)
				h["details"] = details
				for _, detail := range details {
					_, _ = s.DB.Exec("UPDATE t_penjualan_detail SET status_dapur='progress' WHERE id_penjualan=? AND id_item=? AND status_dapur='new'", detail["id_penjualan"], detail["id_item"])
				}
			}
			writeJSON(w, 200, headers)
			return
		}
		if act != "summary" && act != "summary_by_item" {
			invalidAction(w)
			return
		}
		q := "SELECT d.id_item,SUM(d.qty) qty,SUM(d.total) total,mi.item_name,mi.item_code FROM t_penjualan_detail d JOIN m_item mi ON d.id_item=mi.id_item JOIN m_category mc ON mi.category_id=mc.id JOIN t_penjualan t ON d.id_penjualan=t.id_penjualan WHERE mc.print_role=? AND d.status_dapur!='ready' AND t.passcode=? AND t.status='success' AND t.tgl_penjualan=CURDATE()"
		args := []any{d, p}
		if act == "summary_by_item" {
			q += " AND d.id_item=?"
			args = append(args, r.URL.Query().Get("item"))
		}
		q += " GROUP BY d.id_item ORDER BY mi.item_code"
		rows, e := s.Rows(q, args...)
		if e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"summary": rows})
	}
}

func updateDapurItems(s *store.Store, pass string, body map[string]any, status, stockType string, ids []string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	operator := ""
	if o, ok := body["operator"].(map[string]any); ok {
		operator = fmt.Sprint(o["id"])
	}
	for _, id := range ids {
		sale := fmt.Sprint(body["id_penjualan"])
		if list, ok := body["data"].([]any); ok {
			for _, v := range list {
				x, _ := v.(map[string]any)
				if fmt.Sprint(x["id_item"]) == id {
					sale = fmt.Sprint(x["id_penjualan"])
					break
				}
			}
		}
		var qty float64
		var name string
		if err = tx.QueryRow("SELECT d.qty,COALESCE(i.item_name,'') FROM t_penjualan_detail d LEFT JOIN m_item i ON i.id_item=d.id_item WHERE d.id_penjualan=? AND d.id_item=? AND d.passcode=?", sale, id, pass).Scan(&qty, &name); err != nil {
			return err
		}
		res, err := tx.Exec("UPDATE t_penjualan_detail SET status_dapur=?,ready_at=IF(?='ready',NOW(),ready_at),ready_by=IF(?='ready',?,ready_by),reject_at=IF(?='reject',NOW(),reject_at),reject_by=IF(?='reject',?,reject_by) WHERE id_penjualan=? AND id_item=? AND passcode=? AND status_dapur='progress'", status, status, status, operator, status, operator, sale, id, pass)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("detail dapur bukan status progress")
		}
		ings, e := s.Rows("SELECT id_igridient,qty FROM m_item_igridients WHERE id_item=? AND passcode=?", id, pass)
		if e != nil {
			return e
		}
		for _, ing := range ings {
			if _, err = tx.Exec("INSERT INTO t_stock_movement(id_transaction,date,id_igridient,qty_in,qty_out,id_user,type_stock,note,vendor_id,vendor_name,passcode) VALUES(?,CURRENT_DATE(),?,0,?,?,?,?,'','',?)", sale, ing["id_igridient"], nf(ing["qty"])*qty, operator, stockType, name, pass); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
