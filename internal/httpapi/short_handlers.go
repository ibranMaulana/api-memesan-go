package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/memesan/api-react/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func payroll(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch requestValue(r, "act") {
		case "getPayroll":
			rows, e := s.Rows("SELECT id_pay_sallary,FORMAT(sallary,0) sallary,FORMAT(tot_plus,0) tot_plus,FORMAT(tot_minus,0) tot_minus,FORMAT((sallary+tot_plus-tot_minus),0) take_home,DATE_FORMAT(date_rilis,'%d-%m-%Y') date_rilis,MONTHNAME(date_rilis) month FROM pay_sallary WHERE employees_id=? AND date_rilis<=CURRENT_DATE() AND passcode=?", r.URL.Query().Get("employees_id"), r.URL.Query().Get("passcode"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"data": rows})
		case "getPayrollDetail":
			id := r.URL.Query().Get("id_pay_sallary")
			rows, e := s.Rows("SELECT id_pay_sallary,FORMAT(sallary,0) sallary,FORMAT(tot_plus,0) tot_plus,FORMAT(tot_minus,0) tot_minus,FORMAT((sallary+tot_plus-tot_minus),0) take_home,DATE_FORMAT(date_rilis,'%d-%m-%Y') date_rilis,MONTHNAME(date_rilis) month FROM pay_sallary WHERE id_pay_sallary=?", id)
			if e != nil {
				dbError(w, e)
				return
			}
			detail, e := s.Rows("SELECT FORMAT(s.nominal,0) nominal,m.pay_min_name,p.pay_plus_name,s.type FROM pay_sallary_detail s LEFT JOIN pay_min m ON s.pay_id=m.id_pay_min LEFT JOIN pay_plus p ON s.pay_id=p.id_pay_plus WHERE s.id_pay_sallary=?", id)
			if e != nil {
				dbError(w, e)
				return
			}
			for _, v := range detail {
				if v["pay_min_name"] != "" {
					v["pay_name"] = v["pay_min_name"]
				} else {
					v["pay_name"] = v["pay_plus_name"]
				}
				delete(v, "pay_min_name")
				delete(v, "pay_plus_name")
			}
			writeJSON(w, 200, map[string]any{"data": rows, "detail": detail})
		default:
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
		}
	}
}

func siswa(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requestValue(r, "act") != "dataSiswa" {
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
			return
		}
		rows, e := s.Rows("SELECT * FROM mahasiswa")
		if e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"data": rows})
	}
}

func employees(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requestValue(r, "act") != "updateEmployee" {
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
			return
		}
		if err := r.ParseForm(); err != nil {
			writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
			return
		}
		id := r.FormValue("employees_id")
		if id == "" {
			writeJSON(w, 400, map[string]string{"msg": "employees_id dibutuhkan"})
			return
		}
		_, e := s.DB.Exec("UPDATE m_employees SET full_name=?,username=? WHERE employees_id=?", r.FormValue("full_name"), r.FormValue("username"), id)
		if e != nil {
			dbError(w, e)
			return
		}
		if password := r.FormValue("password"); password != "" {
			hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if e != nil {
				dbError(w, e)
				return
			}
			_, e = s.DB.Exec("UPDATE m_employees SET password=? WHERE employees_id=?", string(hash), id)
			if e != nil {
				dbError(w, e)
				return
			}
		}
		writeJSON(w, 200, "success")
	}
}

func salesOmset(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act := requestValue(r, "act")
		p, eid := r.URL.Query().Get("passcode"), r.URL.Query().Get("employees_id")
		q := "SELECT SUM(grand_total) grand_total,SUM(sales_omset) sales_omset,SUM(grand_total)-SUM(sales_omset) pengurang,MONTHNAME(tanggal) month,DATE_FORMAT(tanggal,'%m') bulan FROM a_sales_omset WHERE DATE_FORMAT(tanggal,'%Y')=? AND sales=? AND passcode=?"
		args := []any{time.Now().Format("2006"), eid, p}
		if act == "cekModal" {
			q += " AND DATE_FORMAT(tanggal,'%m')=?"
			args = append(args, r.URL.Query().Get("bulan"))
		}
		q += " GROUP BY MONTH(tanggal),YEAR(tanggal)"
		rows, e := s.Rows(q, args...)
		if e != nil {
			dbError(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"data": rows})
	}
}

func stockOpname(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requestValue(r, "act") != "saveStockOpname" {
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			_ = r.ParseForm()
		}
		p := r.FormValue("passcode")
		items := []map[string]any{}
		if raw := r.FormValue("items"); raw != "" {
			_ = json.Unmarshal([]byte(raw), &items)
		}
		if p == "" || len(items) == 0 {
			writeJSON(w, 400, map[string]string{"msg": "data stock opname dibutuhkan"})
			return
		}
		tx, e := s.DB.Begin()
		if e != nil {
			dbError(w, e)
			return
		}
		defer tx.Rollback()
		id := time.Now().Format("20060102150405")
		diff := 0.0
		for _, item := range items {
			stock := fval(item["stock"])
			physical := fval(item["stock_fisik"])
			in, out := 0.0, 0.0
			if stock >= physical {
				out = stock - physical
			} else {
				in = physical - stock
			}
			diff += out + in
			if _, e = tx.Exec("INSERT INTO t_stock_movement(id_transaction,date,id_item,qty_in,qty_out,id_user,id_branch,passcode) VALUES(?,CURRENT_DATE(),?,?,?,?,?,?)", id, item["id_item"], in, out, r.FormValue("id_user"), r.FormValue("id_branch"), p); e != nil {
				dbError(w, e)
				return
			}
			if _, e = tx.Exec("INSERT INTO t_stock_opname_detail(id_stock_opname,id_item,qty_in,qty_out,stock,stock_fisik,passcode) VALUES(?,?,?,?,?,?,?)", id, item["id_item"], in, out, stock, physical, p); e != nil {
				dbError(w, e)
				return
			}
		}
		_, e = tx.Exec("INSERT INTO t_stock_opname(id_stock_opname,tgl_so,employees_id,tot_diff,id_branch,passcode) VALUES(?,CURRENT_DATE(),?,?,?,?)", id, r.FormValue("id_user"), diff, r.FormValue("id_branch"), p)
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

var _ = strings.TrimSpace
