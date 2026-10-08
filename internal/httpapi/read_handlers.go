package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/memesan/api-react/internal/store"
)

func bank(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("passcode")
		if p == "" {
			writeJSON(w, 200, map[string]any{"status": false, "message": "Passcode is required", "data": nil})
			return
		}
		rows, err := s.Rows("SELECT p.code,p.name FROM m_payment_method p JOIN payment_method_active pa ON p.code=pa.payment_code WHERE pa.passcode=?", p)
		if err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"status": true, "message": "", "data": rows})
	}
}
func notes(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.Rows("SELECT catatan FROM m_catatan WHERE passcode=? ORDER BY sort ASC", r.URL.Query().Get("passcode"))
		if err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
	}
}
func items(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.Rows("SELECT mi.*,cat.print_role FROM m_item mi LEFT JOIN m_category cat ON mi.category_id=cat.id WHERE mi.passcode=? AND mi.status_active='active' AND mi.item_code<>'0' ORDER BY mi.item_code ASC", r.URL.Query().Get("passcode"))
		if err != nil {
			dbError(w, err)
			return
		}
		for _, row := range rows {
			if v, ok := row["decimal"].(string); ok {
				row["decimal"], _ = strconv.Atoi(v)
			}
			if v, ok := row["has_sub_menu"].(string); ok {
				row["has_sub_menu"] = v == "1"
			}
		}
		writeJSON(w, 200, rows)
	}
}
func dashboard(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day, month := time.Now().Format("2006-01-02"), time.Now().Format("2006-01")
		if v, err := time.Parse("02-01-2006", r.URL.Query().Get("tgl")); err == nil {
			day, month = v.Format("2006-01-02"), v.Format("2006-01")
		}
		q := "SELECT COUNT(receipt_number) pesanan,COALESCE(SUM(ppn),0) pajak,COALESCE(SUM(subtotal),0) subtotal,COALESCE(SUM(total),0) total,SUM(IF(type_transaction='Off',1,0)) offline,SUM(IF(type_transaction='Onl',1,0)) online FROM t_penjualan WHERE passcode=? AND tgl_penjualan LIKE ? AND status='success'"
		m, err := s.Rows(q, r.URL.Query().Get("passcode"), month+"%")
		if err != nil {
			dbError(w, err)
			return
		}
		d, err := s.Rows(q, r.URL.Query().Get("passcode"), day+"%")
		if err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"status": "success", "message": "", "data": map[string]any{"params": day, "bulanan": first(m), "harian": first(d)}})
	}
}
func recapList(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var input struct {
			Passcode string `json:"passcode"`
			Page     int    `json:"page"`
		}
		_ = json.Unmarshal(body, &input)
		if input.Passcode == "" {
			writeJSON(w, 400, map[string]string{"msg": "passcode dibutuhkan"})
			return
		}
		offset := input.Page*10 - 10
		if offset < 0 {
			offset = 0
		}
		rows, err := s.Rows("SELECT r.*,e.full_name FROM r_rekap r LEFT JOIN m_employees e ON r.employees_id=e.employees_id WHERE r.passcode=? ORDER BY created_at DESC LIMIT 10 OFFSET ?", input.Passcode, offset)
		if err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, rows)
	}
}
func first(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return map[string]any{"subtotal": 0, "pajak": 0, "total": 0, "pesanan": 0, "offline": 0, "online": 0}
	}
	return rows[0]
}
func dbError(w http.ResponseWriter, err error) {
	writeJSON(w, 500, map[string]string{"error": err.Error()})
}
