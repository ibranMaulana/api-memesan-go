package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/memesan/api-react/internal/store"
)

func bodyMap(r *http.Request) map[string]any {
	b, _ := io.ReadAll(r.Body)
	var v map[string]any
	_ = json.Unmarshal(b, &v)
	return v
}
func text(v any) string { return v.(string) }
func required(w http.ResponseWriter, v any, msg string) bool {
	if v == nil || v == "" {
		writeJSON(w, 400, map[string]string{"msg": msg})
		return false
	}
	return true
}

func category(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act := requestValue(r, "act")
		p := r.URL.Query().Get("passcode")
		switch act {
		case "list":
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT * FROM m_category WHERE passcode=?", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "save":
			d := bodyMap(r)
			if !required(w, d["passcode"], "passcode dibutuhkan") || !required(w, d["name"], "name dibutuhkan") {
				return
			}
			role := "KITCHEN"
			if v, ok := d["print_role"].(string); ok {
				if v == "CASHIER" {
					writeJSON(w, 400, map[string]string{"msg": "CASHIER bukan nilai yang valid untuk print_role"})
					return
				}
				if v == "BAR" {
					role = v
				}
			}
			_, e := s.DB.Exec("INSERT INTO m_category (category,icon,status_active,passcode,print_role) VALUES (?, '', 'active', ?, ?)", d["name"], d["passcode"], role)
			if e != nil {
				dbError(w, e)
				return
			}
			rows, e := s.Rows("SELECT * FROM m_category WHERE passcode=? ORDER BY id DESC LIMIT 1", d["passcode"])
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"status": "success", "data": rows[0]})
		case "update":
			categoryUpdate(w, s, r, false)
		case "delete":
			categoryUpdate(w, s, r, true)
		default:
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
		}
	}
}
func categoryUpdate(w http.ResponseWriter, s *store.Store, r *http.Request, del bool) {
	p, id := r.URL.Query().Get("passcode"), r.URL.Query().Get("id")
	if !required(w, p, "passcode dibutuhkan") || !required(w, id, "id dibutuhkan") {
		return
	}
	rows, e := s.Rows("SELECT * FROM m_category WHERE passcode=? AND id=?", p, id)
	if e != nil {
		dbError(w, e)
		return
	}
	if len(rows) == 0 {
		writeJSON(w, 404, map[string]string{"msg": "category tidak ditemukan"})
		return
	}
	if del {
		_, e = s.DB.Exec("DELETE FROM m_category WHERE passcode=? AND id=?", p, id)
		if e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, map[string]string{"status": "success"})
		return
	}
	d := bodyMap(r)
	if !required(w, d["name"], "name dibutuhkan") {
		return
	}
	_, e = s.DB.Exec("UPDATE m_category SET category=?,print_role=COALESCE(?,print_role) WHERE passcode=? AND id=?", d["name"], d["print_role"], p, id)
	if e != nil {
		dbError(w, e)
		return
	}
	rows, e = s.Rows("SELECT * FROM m_category WHERE passcode=? AND id=?", p, id)
	if e != nil {
		dbError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"status": "success", "data": rows[0]})
}

func meja(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		switch act {
		case "list":
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT id,meja_name AS meja FROM m_meja WHERE passcode=? ORDER BY meja_name ASC", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "save":
			d := bodyMap(r)
			if !required(w, d["passcode"], "passcode dibutuhkan") || !required(w, d["name"], "name dibutuhkan") {
				return
			}
			_, e := s.DB.Exec("INSERT INTO m_meja (meja_name,passcode) VALUES (?,?)", d["name"], d["passcode"])
			if e != nil {
				dbError(w, e)
				return
			}
			rows, e := s.Rows("SELECT * FROM m_meja WHERE passcode=? ORDER BY id DESC LIMIT 1", d["passcode"])
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"status": "success", "data": rows[0]})
		case "update", "delete":
			id := r.URL.Query().Get("id")
			if !required(w, p, "passcode dibutuhkan") || !required(w, id, "id dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT * FROM m_meja WHERE passcode=? AND id=?", p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 404, map[string]string{"msg": "meja tidak ditemukan"})
				return
			}
			if act == "delete" {
				_, e = s.DB.Exec("DELETE FROM m_meja WHERE passcode=? AND id=?", p, id)
				if e != nil {
					dbError(w, e)
					return
				}
				writeJSON(w, 200, map[string]string{"status": "success"})
				return
			}
			d := bodyMap(r)
			if !required(w, d["name"], "name dibutuhkan") {
				return
			}
			_, e = s.DB.Exec("UPDATE m_meja SET meja_name=? WHERE passcode=? AND id=?", d["name"], p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			rows, e = s.Rows("SELECT * FROM m_meja WHERE passcode=? AND id=?", p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"status": "success", "data": rows[0]})
		}
	}
}

func kas(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act, p := requestValue(r, "act"), r.URL.Query().Get("passcode")
		switch act {
		case "list":
			if !required(w, p, "passcode dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT * FROM a_kas WHERE passcode=? ORDER BY log_time DESC", p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
		case "save":
			d := bodyMap(r)
			_, e := s.DB.Exec("INSERT INTO a_kas (nominal,description,tipe,id_branch,passcode,employees_id,log_time) VALUES (?,?,?,'Pusat',?,?,?)", d["nominal"], d["description"], d["tipe"], d["passcode"], d["employees_id"], d["log_time"])
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]string{"status": "success"})
		case "delete":
			id := r.URL.Query().Get("id")
			if !required(w, p, "passcode dibutuhkan") || !required(w, id, "id dibutuhkan") {
				return
			}
			rows, e := s.Rows("SELECT id_kas FROM a_kas WHERE passcode=? AND id_kas=?", p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			if len(rows) == 0 {
				writeJSON(w, 404, map[string]string{"msg": "data kas tidak ditemukan"})
				return
			}
			_, e = s.DB.Exec("DELETE FROM a_kas WHERE passcode=? AND id_kas=?", p, id)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]string{"status": "success"})
		}
	}
}

func customer(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act := requestValue(r, "act")
		if act == "findCustomer" {
			rows, e := s.Rows("SELECT * FROM m_customer WHERE passcode=? AND (nama_customer LIKE ? OR nomor_hp LIKE ?)", r.URL.Query().Get("passcode"), "%"+r.URL.Query().Get("searchCustomer")+"%", "%"+r.URL.Query().Get("searchCustomer")+"%")
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, rows)
			return
		}
		if act == "saveCustomer" {
			d := bodyMap(r)
			if !required(w, d["passcode"], "passcode dibutuhkan") || !required(w, d["name"], "name dibutuhkan") {
				return
			}
			_, e := s.DB.Exec("INSERT INTO m_customer (nama_customer,passcode) VALUES (?,?)", d["name"], d["passcode"])
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
			return
		}
		writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
	}
}

func user(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, 400, map[string]string{"msg": "method tidak diijinkan"})
			return
		}
		d := bodyMap(r)
		if !required(w, d["passcode"], "passcode dibutuhkan") || !required(w, d["employee_id"], "employee_id dibutuhkan") || !required(w, d["pin"], "pin dibutuhkan") || !required(w, d["new_pin"], "new_pin dibutuhkan") {
			return
		}
		rows, e := s.Rows("SELECT employees_id FROM m_employees WHERE employees_id=? AND passcode=? AND pincode=?", d["employee_id"], d["passcode"], d["pin"])
		if e != nil {
			dbError(w, e)
			return
		}
		if len(rows) == 0 {
			writeJSON(w, 404, map[string]string{"msg": "data user tidak ditemukan"})
			return
		}
		_, e = s.DB.Exec("UPDATE m_employees SET pincode=? WHERE employees_id=? AND passcode=?", d["new_pin"], d["employee_id"], d["passcode"])
		if e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, "success")
	}
}

var _ = strconv.Itoa
var _ = text
