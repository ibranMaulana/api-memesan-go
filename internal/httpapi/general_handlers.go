package httpapi

import (
	"net/http"
	"time"

	"github.com/memesan/api-react/internal/store"
)

func umum(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch requestValue(r, "act") {
		case "get":
			p := r.URL.Query().Get("passcode")
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT *,discount_tax_order AS discountTaxOrder FROM m_company WHERE passcode=?", p)
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 404, map[string]string{"msg": "data company tidak ditemukan"})
				return
			}
			writeJSON(w, 200, rows[0])
		case "get_last_counters":
			p := r.URL.Query().Get("passcode")
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(receipt_number,' - ',1),'/',-1) AS UNSIGNED)),0) last_receipt_count,DATE_FORMAT(MAX(created_at),'%Y-%m') last_receipt_month,COALESCE(MAX(CAST(queue_number AS UNSIGNED)),0) last_queue FROM t_penjualan WHERE passcode=? AND DATE(created_at)=CURRENT_DATE()", p)
			if e != nil {
				dbError(w, e)
				return
			}
			row := first(rows)
			month := time.Now().Format("2006-01")
			if v, ok := row["last_receipt_month"].(string); ok && v != "" {
				month = v
			}
			writeJSON(w, 200, map[string]any{"success": true, "data": map[string]any{"last_receipt_count": ival(row["last_receipt_count"]), "last_queue": ival(row["last_queue"]), "last_receipt_month": month}})
		case "check_update":
			version := ival(r.FormValue("version"))
			rows, e := s.Rows("SELECT version,link_app FROM app_version WHERE id=1")
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 500, map[string]string{"msg": "versi aplikasi tidak ditemukan"})
				return
			}
			dbv := ival(rows[0]["version"])
			status, msg := "success", "Sudah versi terbaru"
			if version < dbv {
				status, msg = "update", "Update tersedia"
			}
			if version > dbv {
				_, e = s.DB.Exec("UPDATE app_version SET version=? WHERE id=1", version)
				if e != nil {
					dbError(w, e)
					return
				}
				msg = "Database berhasil diperbarui"
			}
			writeJSON(w, 200, map[string]any{"status": status, "message": msg, "data": map[string]any{"version": map[bool]int{true: version, false: dbv}[version > dbv], "link_app": rows[0]["link_app"]}})
		case "save":
			p := r.URL.Query().Get("passcode")
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			d := bodyMap(r)
			if len(d) == 0 {
				writeJSON(w, 400, map[string]string{"msg": "data dibutuhkan"})
				return
			}
			if !required(w, d["nama_perusahaan"], "data nama_perusahaan dibutuhkan") || !required(w, d["tipe_harga"], "data tipe_harga dibutuhkan") || !required(w, d["tipe_harga_on"], "data tipe_harga_on dibutuhkan") {
				return
			}
			_, e := s.DB.Exec("UPDATE m_company SET nama_perusahaan=?,nomor_hp=?,npwp=?,niop=?,tipe_harga=?,tipe_harga_on=?,up_price=?,showppn=?,taxPercentage=?,rounding=?,discount_tax_order=?,show_queue_number=?,notes=? WHERE passcode=?", d["nama_perusahaan"], d["nomor_hp"], d["npwp"], d["niop"], d["tipe_harga"], d["tipe_harga_on"], d["up_price"], d["showppn"], d["taxPercentage"], ival(d["rounding"]), d["discountTaxOrder"], d["show_queue_number"], d["notes"], p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]string{"status": "success"})
		default:
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
		}
	}
}

func promo(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := requestValue(r, "passcode")
		switch requestValue(r, "act") {
		case "data":
			rows, e := s.Rows("SELECT * FROM m_promo WHERE status_active='active' AND DATE_FORMAT(dari_tanggal,'%Y%m%d')<=DATE_FORMAT(CURRENT_DATE(),'%Y%m%d') AND DATE_FORMAT(sampai_tanggal,'%Y%m%d')>=DATE_FORMAT(CURRENT_DATE(),'%Y%m%d') AND passcode=?", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows})
		case "getData":
			id, branch := r.URL.Query().Get("id_promo"), r.URL.Query().Get("branch")
			rows, e := s.Rows("SELECT * FROM m_promo_detail WHERE id_promo=? AND passcode=?", id, p)
			if e != nil {
				dbError(w, e)
				return
			}
			for _, row := range rows {
				q := "SELECT COALESCE(SUM(qty_in-qty_out),0) FROM t_stock_movement WHERE id_item=? AND passcode=?"
				args := []any{row["id_item"], p}
				if branch != "Pusat" {
					q += " AND id_branch=?"
					args = append(args, branch)
				}
				var stock float64
				_ = s.DB.QueryRow(q, args...).Scan(&stock)
				row["stock"] = stock
			}
			writeJSON(w, 200, rows)
		default:
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
		}
	}
}
