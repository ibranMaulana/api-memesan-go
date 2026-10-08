package httpapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/memesan/api-react/internal/store"
)

func distanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const radius = 6371000.0
	phi1, phi2 := lat1*math.Pi/180, lat2*math.Pi/180
	dphi, dlambda := (lat2-lat1)*math.Pi/180, (lon2-lon1)*math.Pi/180
	a := math.Sin(dphi/2)*math.Sin(dphi/2) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(dlambda/2)*math.Sin(dlambda/2)
	return math.Round(2*radius*math.Atan2(math.Sqrt(a), math.Sqrt(1-a))*100) / 100
}

func absensi(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act := requestValue(r, "act")
		if err := r.ParseForm(); err != nil {
			writeJSON(w, 400, "request tidak valid")
			return
		}
		p, emp := r.FormValue("passcode"), r.FormValue("employees_id")
		today := time.Now().Format("2006-01-02")
		if act == "checkLog" {
			p = r.URL.Query().Get("passcode")
			q := "SELECT e.full_name,l.level_name,e.employees_id,e.photo,a.waktu_checkin,a.lokasi_checkin,a.long_checkin,a.lat_checkin,a.status_checkin,a.waktu_checkout,a.lokasi_checkout,a.long_checkout,a.lat_checkout,a.status_checkout,a.tidak_masuk,a.lampiran,a.keterangan,a.log_time FROM m_employees e JOIN m_level l ON l.level_id=e.level_id LEFT JOIN abs_absensi a ON a.employees_id=e.employees_id AND a.tanggal=CURDATE() AND a.passcode=e.passcode WHERE e.passcode=?"
			args := []any{p}
			if id := r.URL.Query().Get("employees_id"); id != "" {
				q += " AND e.employees_id=?"
				args = append(args, id)
			}
			q += " ORDER BY e.full_name"
			rows, e := s.Rows(q, args...)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows})
			return
		}
		switch act {
		case "checkin":
			_, _ = s.DB.Exec("DELETE FROM abs_absensi WHERE employees_id=? AND tanggal=?", emp, today)
			id := newID("ABS")
			lat, _ := strconv.ParseFloat(r.FormValue("lat_checkin"), 64)
			lon, _ := strconv.ParseFloat(r.FormValue("long_checkin"), 64)
			klat, _ := strconv.ParseFloat(r.FormValue("lat_kantor"), 64)
			klon, _ := strconv.ParseFloat(r.FormValue("long_kantor"), 64)
			_, e := s.DB.Exec("INSERT INTO abs_absensi(id_absensi,employees_id,tanggal,passcode,waktu_checkin,lokasi_checkin,long_checkin,lat_checkin,selisih_waktu_checkin,status_checkin,selisih_jarak_checkin) VALUES(?,?,?,?,CURRENT_TIME(),?,?,?,?,?,?)", id, emp, today, p, r.FormValue("lokasi_checkin"), r.FormValue("long_checkin"), r.FormValue("lat_checkin"), r.FormValue("selisih_waktu_checkin"), attendanceStatus(r, "in"), distanceMeters(klat, klon, lat, lon))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, true)
		case "checkout":
			lat, _ := strconv.ParseFloat(r.FormValue("lat_checkout"), 64)
			lon, _ := strconv.ParseFloat(r.FormValue("long_checkout"), 64)
			klat, _ := strconv.ParseFloat(r.FormValue("lat_kantor"), 64)
			klon, _ := strconv.ParseFloat(r.FormValue("long_kantor"), 64)
			_, e := s.DB.Exec("UPDATE abs_absensi SET waktu_checkout=CURRENT_TIME(),lokasi_checkout=?,long_checkout=?,lat_checkout=?,selisih_waktu_checkout=?,status_checkout=?,selisih_jarak_checkout=? WHERE employees_id=? AND passcode=? AND tanggal=?", r.FormValue("lokasi_checkout"), r.FormValue("long_checkout"), r.FormValue("lat_checkout"), r.FormValue("selisih_waktu_checkout"), attendanceStatus(r, "out"), distanceMeters(klat, klon, lat, lon), emp, p, today)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, true)
		case "checkpoint":
			id := newID("CHP")
			lat, _ := strconv.ParseFloat(r.FormValue("lat_checkpoint"), 64)
			lon, _ := strconv.ParseFloat(r.FormValue("long_checkpoint"), 64)
			klat, _ := strconv.ParseFloat(r.FormValue("lat_kantor"), 64)
			klon, _ := strconv.ParseFloat(r.FormValue("long_kantor"), 64)
			_, e := s.DB.Exec("INSERT INTO abs_checkpoint(id_checkpoint,employees_id,tanggal,waktu_checkpoint,long_checkpoint,lat_checkpoint,lokasi_checkpoint,keterangan_checkpoint,foto_checkpoint,selisih_jarak_checkpoint,passcode) VALUES(?,?,?,CURRENT_TIME(),?,?,?,?,?,?,?)", id, emp, today, r.FormValue("long_checkpoint"), r.FormValue("lat_checkpoint"), r.FormValue("lokasi_checkpoint"), r.FormValue("keterangan_checkpoint"), r.FormValue("foto_checkpoint"), distanceMeters(klat, klon, lat, lon), p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, true)
		case "delCheckpoint":
			_, e := s.DB.Exec("DELETE FROM abs_checkpoint WHERE id_checkpoint=? AND employees_id=? AND passcode=? AND tanggal=?", r.FormValue("id_checkpoint"), emp, p, r.FormValue("tanggal"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, true)
		case "kehadiran":
			rows, e := s.Rows("SELECT *,DATE_FORMAT(tanggal,'%Y%m%d') tgl FROM abs_absensi WHERE employees_id=? AND passcode=? AND DATE_FORMAT(tanggal,'%Y%m')=DATE_FORMAT(CURDATE(),'%Y%m')", r.URL.Query().Get("employees_id"), r.URL.Query().Get("passcode"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows})
		case "statusAbsen":
			rows, e := s.Rows("SELECT waktu_checkin,waktu_checkout,tidak_masuk FROM abs_absensi WHERE employees_id=? AND tanggal=CURDATE()", r.URL.Query().Get("employees_id"))
			if e != nil {
				dbError(w, e)
				return
			}
			row := map[string]any{"checkin": false, "waktu_checkin": "", "checkout": false, "waktu_checkout": "", "tidak_masuk": false}
			if len(rows) > 0 {
				row["checkin"] = rows[0]["waktu_checkin"] != ""
				row["waktu_checkin"] = rows[0]["waktu_checkin"]
				row["checkout"] = rows[0]["waktu_checkout"] != ""
				row["waktu_checkout"] = rows[0]["waktu_checkout"]
				row["tidak_masuk"] = rows[0]["tidak_masuk"] != ""
			}
			writeJSON(w, 200, map[string]any{"data": []any{row}})
		case "tidakMasuk":
			_, _ = s.DB.Exec("DELETE FROM abs_absensi WHERE employees_id=? AND tanggal=?", emp, today)
			id := newID("ABS")
			_, e := s.DB.Exec("INSERT INTO abs_absensi(id_absensi,employees_id,tanggal,passcode,tidak_masuk,lampiran,keterangan) VALUES(?,?,?,?,?,?,?)", id, emp, today, p, r.FormValue("tidak_masuk"), r.FormValue("lampiran"), r.FormValue("keterangan"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, true)
		case "getProduct":
			page := ival(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			rows, e := s.Rows("SELECT id_item id,item_code,item_name,uom,id_rack,type,merk,specification,base_price,net_price,sell_price,category_name,foto_1,foto_2,foto_3,foto_4 FROM m_item WHERE passcode=? LIMIT 5 OFFSET ?", r.URL.Query().Get("passcode"), (page-1)*5)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows, "page": page})
		default:
			invalidAction(w)
		}
	}
}

func attendanceStatus(r *http.Request, typ string) string {
	now := time.Now().Format("15:04:05")
	if typ == "in" {
		if r.FormValue("batas_sebelum_masuk") > now {
			return "Lembur"
		}
		if r.FormValue("batas_keterlambatan_masuk") < now {
			return "Telat"
		}
		return "Sesuai"
	}
	if r.FormValue("batas_sebelum_pulang") > now {
		return "Pulang Cepat"
	}
	if r.FormValue("batas_keterlambatan_pulang") > now {
		return "Sesuai"
	}
	return "Lembur"
}

func requestMoving(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), requestValue(r, "passcode")
		if err := r.ParseForm(); err != nil {
			writeJSON(w, 400, "request tidak valid")
			return
		}
		switch act {
		case "dataCabang":
			rows, e := s.Rows("SELECT id_branch,branch_name FROM m_branch WHERE passcode=?", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "detailMoving":
			rows, e := s.Rows("SELECT r.id_rm,r.id_item id,r.qty_req,r.qty_apply,r.qty_receive,r.description,r.passcode,i.item_name,i.category_name,i.foto_1 FROM p_request_moving_detail r JOIN m_item i ON i.id_item=r.id_item WHERE r.id_rm=?", requestValue(r, "xid"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"detail": rows})
		case "dataMoving":
			page := ival(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			rows, e := s.Rows("SELECT b.branch_name,rm.* FROM p_request_moving rm JOIN m_branch b ON b.id_branch=rm.id_branch WHERE rm.passcode=? ORDER BY rm.id_rm DESC LIMIT 9 OFFSET ?", p, (page-1)*9)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows, "page": page})
		case "request":
			items := parseItems(r.FormValue("items"))
			if len(items) == 0 {
				writeJSON(w, 400, "items diperlukan")
				return
			}
			id := newID("MOV")
			tx, e := s.DB.Begin()
			if e != nil {
				dbError(w, e)
				return
			}
			defer tx.Rollback()
			total := 0.0
			for _, i := range items {
				q := fval(i["qty"])
				total += q
				if _, e = tx.Exec("INSERT INTO p_request_moving_detail(id_rm,id_item,qty_req,passcode) VALUES(?,?,?,?)", id, i["id"], q, p); e != nil {
					dbError(w, e)
					return
				}
			}
			_, e = tx.Exec("INSERT INTO p_request_moving(id_rm,id_branch,create_date,tgl_req,admin_req,tot_qty_req,memo_req,passcode) VALUES(?,?,CURRENT_DATE(),?,?,?, ?,?)", id, r.FormValue("id_branch"), mysqlDate(r.FormValue("date")), r.FormValue("admin_req"), total, r.FormValue("memo_req"), p)
			if e != nil {
				dbError(w, e)
				return
			}
			if e = tx.Commit(); e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, true)
		case "apply", "receive":
			movingUpdate(w, s, r, act, p)
		default:
			invalidAction(w)
		}
	}
}

func parseItems(raw string) []map[string]any {
	var v []map[string]any
	_ = json.Unmarshal([]byte(raw), &v)
	return v
}

func dapurRekap(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p, d := requestValue(r, "act"), requestValue(r, "passcode"), r.URL.Query().Get("dapur")
		if act == "summary" {
			rows, e := s.Rows("SELECT d.id_item,SUM(d.qty) qty,SUM(d.total) total,mi.item_name,mi.item_code FROM t_penjualan_detail d JOIN m_item mi ON mi.id_item=d.id_item JOIN m_category mc ON mc.id=mi.category_id JOIN t_penjualan t ON t.id_penjualan=d.id_penjualan WHERE mc.print_role=? AND d.status_dapur!='ready' AND t.passcode=? AND t.status='success' GROUP BY d.id_item ORDER BY mi.item_code", d, p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"summary": rows})
			return
		}
		if act == "ready" || act == "ready_bulk" {
			body := bodyMap(r)
			if act == "ready" {
				for _, id := range strings.Split(fmt.Sprint(body["id_items"]), ",") {
					_, _ = s.DB.Exec("UPDATE t_penjualan_detail SET status_dapur='ready' WHERE id_penjualan=? AND id_item=? AND passcode=?", body["id_penjualan"], id, p)
				}
			}
			writeJSON(w, 200, true)
			return
		}
		if act == "list" || act == "list_by_items" {
			q := "SELECT p.id_penjualan,p.nomor_transaksi,p.receipt_number,p.id_branch,p.passcode,p.keterangan,p.meja,p.tgl_penjualan,p.customer,IFNULL(c.customer_name,'') customer_name,p.sales,p.user,p.grand_total total,p.payment_amount,p.status,p.created_at FROM t_penjualan p LEFT JOIN m_customer c ON p.customer=c.customer_code WHERE p.passcode=? AND p.status='success'"
			args := []any{p}
			if act == "list_by_items" {
				q += " AND EXISTS(SELECT 1 FROM t_penjualan_detail d JOIN m_item i ON i.id_item=d.id_item JOIN m_category mc ON mc.id=i.category_id WHERE d.id_penjualan=p.id_penjualan AND d.id_item=? AND mc.print_role=? AND d.status_dapur!='ready')"
				args = append(args, r.URL.Query().Get("item"), d)
			} else {
				q += " AND EXISTS(SELECT 1 FROM t_penjualan_detail d JOIN m_item i ON i.id_item=d.id_item JOIN m_category mc ON mc.id=i.category_id WHERE d.id_penjualan=p.id_penjualan AND mc.print_role=? AND d.status_dapur='new')"
				args = append(args, d)
			}
			q += " ORDER BY p.nomor_transaksi ASC LIMIT 5"
			rows, e := s.Rows(q, args...)
			if e != nil {
				dbError(w, e)
				return
			}
			for _, tr := range rows {
				details, _ := s.Rows("SELECT d.id_penjualan_detail,d.id_penjualan,d.nomor_transaksi,d.catatan,d.id_item,d.qty,d.total,mi.item_name FROM t_penjualan_detail d JOIN m_item mi ON mi.id_item=d.id_item JOIN m_category mc ON mc.id=mi.category_id WHERE d.id_penjualan=? AND mc.print_role=? AND d.status_dapur!='ready'", tr["id_penjualan"], d)
				tr["details"] = details
				_, _ = s.DB.Exec("UPDATE t_penjualan_detail SET status_dapur='progress' WHERE id_penjualan=?", tr["id_penjualan"])
			}
			writeJSON(w, 200, rows)
			return
		}
		invalidAction(w)
	}
}

func movingUpdate(w http.ResponseWriter, s *store.Store, r *http.Request, act, passcode string) {
	id := r.FormValue("id")
	if id == "" {
		writeJSON(w, 400, "id diperlukan")
		return
	}
	items := parseItems(r.FormValue("items"))
	if len(items) == 0 {
		writeJSON(w, 400, "items diperlukan")
		return
	}
	tx, e := s.DB.Begin()
	if e != nil {
		dbError(w, e)
		return
	}
	defer tx.Rollback()
	typ, branch, admin := "Moving", r.FormValue("id_branch"), r.FormValue("admin_apply")
	if act == "receive" {
		typ = "Supply Moving"
		branch = "Pusat"
		admin = r.FormValue("admin_receive")
	}
	_, e = tx.Exec("DELETE FROM t_stock_movement WHERE id_transaction=? AND type_stock=?", id, typ)
	if e != nil {
		dbError(w, e)
		return
	}
	total := 0.0
	for _, item := range items {
		q := fval(item["qty"])
		total += q
		field := "qty_apply"
		if act == "receive" {
			field = "qty_receive"
		}
		_, e = tx.Exec("UPDATE p_request_moving_detail SET "+field+"=? WHERE id_rm=? AND id_item=? AND passcode=?", q, id, item["id"], passcode)
		if e != nil {
			dbError(w, e)
			return
		}
		in, out := 0.0, q
		if act == "receive" {
			in, out = q, 0
		}
		_, e = tx.Exec("INSERT INTO t_stock_movement(id_transaction,date,id_item,qty_in,qty_out,id_user,id_branch,type_stock,note,passcode) VALUES(?,CURRENT_DATE(),?,?,?,?,?,?,?,?)", id, item["id"], in, out, admin, branch, typ, typ, passcode)
		if e != nil {
			dbError(w, e)
			return
		}
	}
	col, adminCol, dateCol, memoCol := "tot_qty_apply", "admin_apply", "tgl_apply", "memo_apply"
	if act == "receive" {
		col, adminCol, dateCol, memoCol = "tot_qty_receive", "admin_receive", "tgl_receive", "memo_receive"
	}
	_, e = tx.Exec("UPDATE p_request_moving SET "+col+"=?,"+adminCol+"=?,"+dateCol+"=?,"+memoCol+"=? WHERE id_rm=? AND passcode=?", total, admin, r.FormValue("date"), r.FormValue("memo_apply"), id, passcode)
	if e != nil {
		dbError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		dbError(w, e)
		return
	}
	writeJSON(w, 200, true)
}

func supply(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), requestValue(r, "passcode")
		if r.Method == "POST" {
			if e := r.ParseForm(); e != nil {
				writeJSON(w, 400, "request tidak valid")
				return
			}
			p = r.FormValue("passcode")
			if act == "request" {
				items := parseItems(r.FormValue("items"))
				if len(items) == 0 {
					writeJSON(w, 400, "items diperlukan")
					return
				}
				id := newID("SPL")
				tx, e := s.DB.Begin()
				if e != nil {
					dbError(w, e)
					return
				}
				defer tx.Rollback()
				tot := 0.0
				for _, i := range items {
					q := fval(i["qty"])
					tot += q
					if _, e = tx.Exec("INSERT INTO p_request_supply_detail(id_rs,id_item,qty_req,passcode) VALUES(?,?,?,?)", id, i["id"], q, p); e != nil {
						dbError(w, e)
						return
					}
				}
				var admin string
				_ = s.DB.QueryRow("SELECT employees_id FROM m_employees WHERE full_name=? AND passcode=?", r.FormValue("admin"), p).Scan(&admin)
				_, e = tx.Exec("INSERT INTO p_request_supply(id_rs,id_branch,tgl_req,admin_req,memo_req,tot_qty_req,create_date,passcode) VALUES(?,?,?, ?,?,?,CURRENT_DATE(),?)", id, r.FormValue("id_branch"), mysqlDate(r.FormValue("date")), admin, r.FormValue("description"), tot, p)
				if e != nil {
					dbError(w, e)
					return
				}
				if e = tx.Commit(); e != nil {
					dbError(w, e)
					return
				}
				writeJSON(w, 200, true)
				return
			}
			if act == "apply" || act == "receive" {
				supplyUpdate(w, s, r, act, p)
				return
			}
		}
		if id := r.URL.Query().Get("id"); id != "" {
			rows, e := s.Rows("SELECT s.*,b.branch_name FROM p_request_supply s LEFT JOIN m_branch b ON s.id_branch=b.id_branch WHERE s.id_rs=?", id)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows})
			return
		}
		rows, e := s.Rows("SELECT s.*,b.branch_name FROM p_request_supply s LEFT JOIN m_branch b ON s.id_branch=b.id_branch WHERE s.passcode=? ORDER BY s.id_rs DESC", p)
		if e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"data": rows})
	}
}

func supplyUpdate(w http.ResponseWriter, s *store.Store, r *http.Request, act, p string) {
	id := r.FormValue("id")
	items := parseItems(r.FormValue("items"))
	if id == "" || len(items) == 0 {
		writeJSON(w, 400, "id dan items diperlukan")
		return
	}
	tx, e := s.DB.Begin()
	if e != nil {
		dbError(w, e)
		return
	}
	defer tx.Rollback()
	typ, branch, field, adminCol, dateCol, memoCol := "Supply", "Pusat", "qty_apply", "admin_apply", "tgl_apply", "memo_apply"
	if act == "receive" {
		typ, field, branch, adminCol, dateCol, memoCol = "Supply Receive", "qty_receive", "", "admin_receive", "tgl_receive", "memo_receive"
	}
	_, e = tx.Exec("DELETE FROM t_stock_movement WHERE id_transaction=? AND type_stock=?", id, typ)
	if e != nil {
		dbError(w, e)
		return
	}
	tot := 0.0
	for _, i := range items {
		q := fval(i["qty"])
		if q == 0 {
			continue
		}
		tot += q
		if _, e = tx.Exec("UPDATE p_request_supply_detail SET "+field+"=? WHERE id_rs=? AND id_item=? AND passcode=?", q, id, i["id"], p); e != nil {
			dbError(w, e)
			return
		}
		in, out := 0.0, q
		if act == "receive" {
			in, out = q, 0
			var b string
			_ = s.DB.QueryRow("SELECT id_branch FROM p_request_supply WHERE id_rs=?", id).Scan(&b)
			branch = b
		}
		if _, e = tx.Exec("INSERT INTO t_stock_movement(id_transaction,date,id_item,qty_in,qty_out,id_user,id_branch,type_stock,note,passcode) VALUES(?,CURRENT_DATE(),?,?,?,?,?,?,?,?)", id, i["id"], in, out, r.FormValue("admin"), branch, typ, typ, p); e != nil {
			dbError(w, e)
			return
		}
	}
	if _, e = tx.Exec("UPDATE p_request_supply SET "+dateCol+"=?,"+adminCol+"=?,"+memoCol+"=?,"+map[bool]string{true: "tot_qty_receive", false: "tot_qty_apply"}[act == "receive"]+"=? WHERE id_rs=? AND passcode=?", r.FormValue("date"), r.FormValue("admin"), r.FormValue("description"), tot, id, p); e != nil {
		dbError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		dbError(w, e)
		return
	}
	writeJSON(w, 200, true)
}

func draftPenjualan(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act := requestValue(r, "act")
		if act == "list" || act == "listDraft" {
			rows, e := s.Rows("SELECT p.*,IFNULL(c.customer_name,'') customer_name FROM t_penjualan p LEFT JOIN m_customer c ON p.customer=c.customer_code WHERE p.passcode=? AND p.status='draft' ORDER BY p.updated_at DESC,p.created_at DESC", requestValue(r, "passcode"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
			return
		}
		if act == "detail" || act == "getDraft" {
			rows, e := s.Rows("SELECT p.*,IFNULL(c.customer_name,'') customer_name FROM t_penjualan p LEFT JOIN m_customer c ON p.customer=c.customer_code WHERE p.passcode=? AND p.status='draft' AND (p.id_penjualan=? OR p.meja=?) LIMIT 1", requestValue(r, "passcode"), requestValue(r, "id_penjualan"), requestValue(r, "meja"))
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 200, map[string]any{"status": false, "msg": "Draft tidak ditemukan", "data": nil})
				return
			}
			details, _ := s.Rows("SELECT d.*,IF(d.item_name IS NULL OR d.item_name='',i.item_name,d.item_name) item_name FROM t_penjualan_detail d LEFT JOIN m_item i ON d.id_item=i.id_item WHERE d.id_penjualan=? AND d.passcode=?", rows[0]["id_penjualan"], requestValue(r, "passcode"))
			rows[0]["details"] = details
			writeJSON(w, 200, map[string]any{"status": true, "data": rows[0]})
			return
		}
		if act == "delete" || act == "deleteDraft" {
			p, id := requestValue(r, "passcode"), requestValue(r, "id_penjualan")
			tx, e := s.DB.Begin()
			if e != nil {
				dbError(w, e)
				return
			}
			defer tx.Rollback()
			if _, e = tx.Exec("DELETE FROM t_penjualan_detail WHERE id_penjualan=? AND passcode=?", id, p); e != nil {
				dbError(w, e)
				return
			}
			res, e := tx.Exec("DELETE FROM t_penjualan WHERE id_penjualan=? AND passcode=? AND status='draft'", id, p)
			if e != nil {
				dbError(w, e)
				return
			}
			n, _ := res.RowsAffected()
			if e = tx.Commit(); e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"status": n > 0})
			return
		}
		if act == "save" || act == "saveDraft" {
			var d map[string]any
			if json.NewDecoder(r.Body).Decode(&d) != nil {
				writeJSON(w, 400, map[string]any{"status": false, "msg": "Payload kosong"})
				return
			}
			p := fmt.Sprint(d["passcode"])
			if p == "" {
				writeJSON(w, 400, map[string]any{"status": false, "msg": "Passcode dibutuhkan"})
				return
			}
			items := parseItems(fmt.Sprint(d["items"]))
			if len(items) == 0 {
				items = parseItems(fmt.Sprint(d["item"]))
			}
			if len(items) == 0 {
				writeJSON(w, 400, map[string]any{"status": false, "msg": "items diperlukan"})
				return
			}
			id := fmt.Sprint(d["id_penjualan"])
			if id == "" || id == "<nil>" {
				id = newID("PNJ")
			}
			tx, e := s.DB.Begin()
			if e != nil {
				dbError(w, e)
				return
			}
			defer tx.Rollback()
			_, e = tx.Exec("DELETE FROM t_penjualan_detail WHERE id_penjualan=? AND passcode=?", id, p)
			if e != nil {
				dbError(w, e)
				return
			}
			total := 0.0
			for _, i := range items {
				q := fval(i["qty"])
				price := fval(i["price"])
				total += q * price
				_, e = tx.Exec("INSERT INTO t_penjualan_detail(id_penjualan,id_item,item_name,qty,net_price,total,passcode,type_transaction) VALUES(?,?,?,?,?,?,?,?)", id, i["id"], i["name"], q, price, q*price, p, "Offline")
				if e != nil {
					dbError(w, e)
					return
				}
			}
			_, e = tx.Exec("INSERT INTO t_penjualan(id_penjualan,passcode,id_branch,meja,tgl_penjualan,subtotal,grand_total,status,type_transaction,created_at) VALUES(?,?,?,?,CURRENT_DATE(),?,?, 'draft','Offline',NOW()) ON DUPLICATE KEY UPDATE subtotal=VALUES(subtotal),grand_total=VALUES(grand_total),updated_at=NOW()", id, p, d["id_branch"], d["meja"], total, total)
			if e != nil {
				dbError(w, e)
				return
			}
			if e = tx.Commit(); e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"status": true, "is_update": true, "trans": []string{id}, "data": map[string]any{"id_penjualan": id, "status": "draft"}})
			return
		}
		invalidAction(w)
	}
}
