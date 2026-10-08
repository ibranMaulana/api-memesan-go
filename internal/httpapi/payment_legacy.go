package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/memesan/api-react/internal/store"
)

func pembayaran(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		switch act {
		case "list":
			page := ival(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			date := r.URL.Query().Get("tgl")
			if date == "" {
				date = "CURDATE()"
			} else {
				date = "?"
			}
			q := "SELECT p.id_penjualan,p.nomor_transaksi,p.receipt_number,p.id_branch,p.passcode,p.keterangan,p.meja,p.tgl_penjualan,p.customer,IFNULL(c.customer_name,'') customer_name,p.sales,p.user,p.payment_method,p.subtotal,p.discount,p.total_upprice service,p.total_net,p.nama_tamu,p.total_ppn,p.donation,p.grand_total total,p.payment_amount,p.tipe_harga,p.type_transaction,p.total_discount,p.ppn,p.status,p.created_at,p.updated_at,p.cancel_by,p.dine_in,p.queue_number FROM t_penjualan p LEFT JOIN m_customer c ON p.customer=c.customer_code WHERE p.passcode=? AND p.tgl_penjualan=" + date + " ORDER BY p.created_at DESC LIMIT 10 OFFSET ?"
			args := []any{p}
			if date == "?" {
				args = append(args, r.URL.Query().Get("tgl"))
			}
			args = append(args, (page-1)*10)
			rows, e := s.Rows(q, args...)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "cancel":
			var d map[string]any
			if json.NewDecoder(r.Body).Decode(&d) != nil {
				writeJSON(w, 400, "request tidak valid")
				return
			}
			id := d["id"]
			var n int
			_ = s.DB.QueryRow("SELECT COUNT(*) FROM refund_transaction WHERE original_transaction_id=?", id).Scan(&n)
			if n > 0 {
				writeJSON(w, 200, "Tidak dapat membatalkan transaksi yang sudah memiliki refund")
				return
			}
			res, e := s.DB.Exec("UPDATE t_penjualan SET status='cancel',cancel_by=? WHERE id_penjualan=? AND passcode=?", d["cancelBy"], id, d["passcode"])
			if e != nil {
				dbError(w, e)
				return
			}
			affected, _ := res.RowsAffected()
			writeJSON(w, 200, affected > 0)
		case "cekid":
			rows, e := s.Rows("SELECT id_penjualan FROM t_penjualan WHERE passcode=? ORDER BY id_penjualan DESC LIMIT 1", p)
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 200, "")
			} else {
				writeJSON(w, 200, rows[0]["id_penjualan"])
			}
		case "savePembayaran", "savePembayaranNonProduk":
			var d map[string]any
			if json.NewDecoder(r.Body).Decode(&d) != nil {
				writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
				return
			}
			ids, err := savePayments(s, d, act == "savePembayaranNonProduk")
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

// savePayments mirrors the two PHP save actions.  One transaction covers each
// cart, so a missing product, submenu, detail, or split payment cannot leave a
// sale header behind.
func savePayments(s *store.Store, data map[string]any, nonProduct bool) ([]string, error) {
	pass := str(data["passcode"])
	if pass == "" {
		return nil, fmt.Errorf("passcode dibutuhkan")
	}
	branch := data["id_branch"]
	raw, ok := data["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("data dibutuhkan")
	}
	priceType := str(data["tipe_harga"])
	onlineType := str(data["tipe_harga_on"])
	if onlineType == "" {
		onlineType = priceType
	}
	tax := 10.0
	var taxValue sql.NullFloat64
	if err := s.DB.QueryRow("SELECT taxPercentage FROM m_company WHERE passcode=? LIMIT 1", pass).Scan(&taxValue); err == nil && taxValue.Valid {
		tax = taxValue.Float64
	}
	ids := make([]string, 0, len(raw))
	for _, v := range raw {
		tr, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("transaksi tidak valid")
		}
		tx, err := s.DB.Begin()
		if err != nil {
			return nil, err
		}
		committed := false
		func() {
			defer func() {
				if !committed {
					_ = tx.Rollback()
				}
			}()
			date := str(tr["date"])
			if date == "" {
				date = ""
			}
			receipt := str(tr["receiptNumber"])
			if nonProduct && receipt != "" {
				var old string
				if e := tx.QueryRow("SELECT id_penjualan FROM t_penjualan WHERE passcode=? AND receipt_number=? LIMIT 1", pass, receipt).Scan(&old); e == nil {
					ids = append(ids, old)
					committed = true
					return
				}
			}
			id := newID("PNJ")
			number := id
			if receipt != "" {
				number = receipt
			}
			if !nonProduct {
				if old := str(tr["date"]) + " " + str(tr["time"]); old != " " {
					var duplicate string
					if e := tx.QueryRow("SELECT id_penjualan FROM t_penjualan WHERE passcode=? AND created_at=? LIMIT 1", pass, old).Scan(&duplicate); e == nil {
						ids = append(ids, duplicate)
						committed = true
						return
					}
				}
			}
			items, _ := tr["item"].([]any)
			subtotal, dpp, ppn, service, nonTaxTotal := 0.0, 0.0, 0.0, 0.0, 0.0
			for _, iv := range items {
				item, _ := iv.(map[string]any)
				qty := num(item["qty"])
				if qty == 0 {
					qty = 1
				}
				itemID := str(item["id"])
				if itemID == "" {
					itemID = str(item["id_item"])
				}
				name := str(item["name"])
				price := num(item["price"])
				onlinePrice := num(item["net_price_on"])
				nonTax := 0
				if !nonProduct {
					var dbPrice, dbOnline float64
					var nt int
					e := tx.QueryRow("SELECT net_price,net_price_on,non_tax FROM m_item WHERE id_item=? AND passcode=? AND status_active='active'", itemID, pass).Scan(&dbPrice, &dbOnline, &nt)
					if e != nil {
						returnErr(&err, e)
						return
					}
					price = dbPrice
					onlinePrice = dbOnline
					nonTax = nt
				}
				disc := num(item["discountAmount"])
				if disc == 0 {
					disc = num(item["discount"])
				}
				if pct := num(item["discountPercentage"]); pct > 0 {
					disc = price * pct / 100
				}
				if disc > price {
					disc = price
				}
				delta := 0.0
				subID := any(nil)
				subName := ""
				if !nonProduct {
					subID = item["selectedSubMenuId"]
					if subID == nil {
						subID = item["sub_menu_id"]
					}
					if subID != nil && str(subID) != "" {
						var sid int
						var add float64
						if e := tx.QueryRow("SELECT id, name, additional_price FROM sub_menu_options WHERE id=? AND passcode=? AND status_active='active'", subID, pass).Scan(&sid, &subName, &add); e != nil {
							returnErr(&err, fmt.Errorf("Pilihan submenu tidak ditemukan"))
							return
						}
						delta = add
						subID = sid
					}
				}
				effective := max0(price-disc) + delta
				line := effective * qty
				taxable := nonTax == 0
				linePPN := 0.0
				if taxable {
					if str(priceType) == "include" && !nonProduct {
						base := line / (1 + tax/100)
						linePPN = line - base
						dpp += base
					} else {
						linePPN = round0(line * tax / 100)
						dpp += line
					}
				}
				subtotal += line
				ppn += linePPN
				service += num(item["service"]) * qty
				if nonTax == 1 {
					nonTaxTotal += line
				}
				_, e := tx.Exec("INSERT INTO t_penjualan_detail(id_penjualan,nomor_transaksi,catatan,id_item,item_name,qty,net_price,sub_menu_id,sub_menu_name,sub_menu_price_delta,ppn,price_after_ppn,discount_nominal,up_price,net_price_on,ppn_on,price_after_ppn_on,show_price,show_price_off,dpp,total,passcode,type_transaction,nama_tamu,non_tax,hpp,hpp_on) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, number, str(item["catatan"]), itemID, name, qty, price, subID, subName, delta, linePPN/qty, effective+linePPN/qty, disc, num(item["service"]), onlinePrice, 0, onlinePrice, num(item["show_price"]), num(item["show_price_off"]), func() float64 {
					if taxable {
						return line - linePPN
					}
					return 0
				}(), line, pass, str(tr["transactionType"]), str(item["namaTamu"]), nonTax, num(item["hpp"])*qty, num(item["hpp_on"])*qty)
				if e != nil {
					returnErr(&err, e)
					return
				}
			}
			status := str(tr["status"])
			if str(tr["cancelBy"]) == "" {
				status = "success"
			}
			total := num(tr["total"])
			if total == 0 {
				total = subtotal - num(tr["discount"])
			}
			customer := ""
			if c, ok := tr["customer"].(map[string]any); ok {
				customer = str(c["id"])
			}
			operator := ""
			user := ""
			if o, ok := tr["operator"].(map[string]any); ok {
				operator = str(o["fullname"])
				user = str(o["id"])
			}
			_, err = tx.Exec("INSERT INTO t_penjualan(id_penjualan,nomor_transaksi,receipt_number,id_branch,passcode,meja,tgl_penjualan,customer,sales,user,payment_method,subtotal,discount,total_discount,ppn,total,donation,payment_amount,created_at,status,cancel_by,type_transaction,tipe_harga,total_upprice,total_net,total_ppn,total_non_tax,grand_total,dine_in,nama_tamu,service,queue_number,data_source) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, number, receipt, branch, pass, str(tr["table"]), date, customer, operator, user, str(tr["paymentMethode"]), subtotal, num(tr["discount"]), num(tr["totalDiscount"]), ppn, subtotal-num(tr["discount"]), num(tr["donation"]), num(tr["paymentAmount"]), date+" "+str(tr["time"]), status, str(tr["cancelBy"]), func() string {
				if nonProduct {
					return "Off"
				}
				return str(tr["transactionType"])
			}(), str(priceType)[:min(1, len(str(priceType)))], service, dpp, ppn, nonTaxTotal, total, str(tr["dineIn"]), str(tr["namaTamu"]), str(data["with_service"]), tr["queue_number"], "api-react")
			if err != nil {
				return
			}
			if str(tr["paymentMethode"]) == "SPLIT" {
				if ps, ok := tr["payments"].([]any); ok {
					for _, pv := range ps {
						pm, _ := pv.(map[string]any)
						if _, err = tx.Exec("INSERT INTO penjualan_payments(id_penjualan,method,amount,received_amount,change_amount,passcode) VALUES(?,?,?,?,?,?)", id, pm["method"], pm["amount"], pm["receivedAmount"], pm["changeAmount"], pass); err != nil {
							return
						}
					}
				}
			}
			err = tx.Commit()
			if err == nil {
				committed = true
				ids = append(ids, id)
			}
		}()
		if err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func returnErr(dst *error, e error) { *dst = e }
func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func num(v any) float64 { f, _ := strconv.ParseFloat(str(v), 64); return f }
func max0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}
func round0(v float64) float64 {
	if v >= 0 {
		return float64(int(v + 0.5))
	}
	return float64(int(v - 0.5))
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func gaji(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requestValue(r, "act") == "value" {
			writeJSON(w, 200, nil)
			return
		}
		invalidAction(w)
	}
}
