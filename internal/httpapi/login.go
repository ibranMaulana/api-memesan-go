package httpapi

import (
	"net/http"

	"github.com/memesan/api-react/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func login(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"msg": "request tidak valid"})
			return
		}
		act := requestValue(r, "act")
		username, password := r.FormValue("username"), r.FormValue("password")
		if act == "saveUserRegister" {
			p := r.FormValue("passcode")
			rows, _ := s.Rows("SELECT passcode FROM m_company WHERE passcode=?", p)
			if len(rows) == 0 {
				writeJSON(w, 200, "Maaf Passcode Salah !")
				return
			}
			var exists int
			_ = s.DB.QueryRow("SELECT COUNT(*) FROM m_employees WHERE username=? OR email_personal=? OR nomor_hp=?", username, r.FormValue("email_personal"), r.FormValue("nomor_hp")).Scan(&exists)
			if exists > 0 {
				writeJSON(w, 200, "Maaf email sudah terdaftar")
				return
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			_, e := s.DB.Exec("INSERT INTO m_employees(employees_id,full_name,email_personal,nomor_hp,username,password,passcode) VALUES(?,?,?,?,?,?,?)", newID("EMP"), r.FormValue("full_name"), r.FormValue("email_personal"), r.FormValue("nomor_hp"), username, string(hash), p)
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
			return
		}
		if act == "verifikasiSMS" {
			var code string
			_ = s.DB.QueryRow("SELECT passcode_sms FROM m_employees WHERE employees_id=?", r.FormValue("employees_id")).Scan(&code)
			if code == r.FormValue("passcode_sms") {
				writeJSON(w, 200, "success")
			} else {
				writeJSON(w, 200, "Maaf passcode salah !")
			}
			return
		}
		if act == "pincodeSave" {
			_, e := s.DB.Exec("UPDATE m_employees SET pincode=? WHERE employees_id=?", r.FormValue("pincode"), r.FormValue("employees_id"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
			return
		}
		if act == "getPincode" {
			var code string
			_ = s.DB.QueryRow("SELECT pincode FROM m_employees WHERE pincode=? AND employees_id=? AND passcode=?", r.URL.Query().Get("pincode"), r.URL.Query().Get("employees_id"), r.URL.Query().Get("passcode")).Scan(&code)
			if code != "" {
				writeJSON(w, 200, "success")
			} else {
				writeJSON(w, 200, "Maaf pincode salah")
			}
			return
		}
		if act == "saveCompanyRegister" {
			if username == "" || password == "" || r.FormValue("email") == "" {
				writeJSON(w, 400, map[string]string{"msg": "data registrasi dibutuhkan"})
				return
			}
			var existing int
			_ = s.DB.QueryRow("SELECT COUNT(*) FROM m_employees WHERE (email_personal=? OR nomor_hp=? OR username=?) AND status_active='active'", r.FormValue("email"), r.FormValue("nomor_hp"), username).Scan(&existing)
			if existing > 0 {
				writeJSON(w, 200, "Maaf Data sudah terdaftar")
				return
			}
			tx, e := s.DB.Begin()
			if e != nil {
				dbError(w, e)
				return
			}
			defer tx.Rollback()
			pass := newID("CMP")
			companyID := newID("CMY")
			empID := newID("EMP")
			levelID := newID("LVL")
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			_, e = tx.Exec("INSERT INTO m_company(id_company,nama_perusahaan,email_perusahaan,bidang_usaha,nama_lengkap,nomor_hp,passcode,employees_id) VALUES(?,?,?,?,?,?,?,?)", companyID, r.FormValue("nama_perusahaan"), r.FormValue("email_perusahaan"), r.FormValue("bidang_usaha"), r.FormValue("nama_lengkap"), r.FormValue("nomor_hp"), pass, empID)
			if e != nil {
				dbError(w, e)
				return
			}
			_, e = tx.Exec("INSERT INTO m_level(level_id,level_name,status_delete,passcode) VALUES(?,'Owner',0,?)", levelID, pass)
			if e != nil {
				dbError(w, e)
				return
			}
			_, e = tx.Exec("INSERT INTO m_employees(employees_id,full_name,level_id,username,email_personal,password,nomor_hp,passcode,owner,verified) VALUES(?,?,?,?,?,?,?,?, 'Yes','Yes')", empID, r.FormValue("nama_lengkap"), levelID, username, r.FormValue("email_personal"), string(hash), r.FormValue("nomor_hp"), pass)
			if e != nil {
				dbError(w, e)
				return
			}
			if e = tx.Commit(); e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
			return
		}
		if username == "" {
			writeJSON(w, 200, "Masukkan username dan password !")
			return
		}
		rows, err := s.Rows(`SELECT e.passcode,e.password,e.full_name,e.nomor_hp,e.type_branch,e.nip,l.level_id,l.level_name,SUBSTRING_INDEX(l.level_name,' ',1) type_level,e.verified,e.photo,e.employees_id,e.pincode FROM m_employees e LEFT JOIN m_level l ON e.level_id=l.level_id WHERE e.username=? AND e.status_active='active'`, username)
		if err != nil {
			dbError(w, err)
			return
		}
		if len(rows) == 0 {
			writeJSON(w, 200, "Username tidak tersedia!")
			return
		}
		employee := rows[0]
		hash, _ := employee["password"].(string)
		if password == "" {
			writeJSON(w, 200, "Masukkan Password !")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			writeJSON(w, 200, "Password Salah !")
			return
		}
		if employee["verified"] != "Yes" {
			writeJSON(w, 200, "Maaf data anda belum terverifikasi, silahkan hubungi admin !")
			return
		}
		passcode, _ := employee["passcode"].(string)
		company, err := s.Rows(`SELECT id_company,nama_perusahaan,bidang_usaha,nomor_hp,alamat,logo_perusahaan,jumlah_meja,tipe_harga,tipe_harga_on,grand_ppn_on,grand_ppn_off,print,notes,service,discount_tax_order AS discountTaxOrder,url_domain FROM m_company WHERE passcode=?`, passcode)
		if err != nil {
			dbError(w, err)
			return
		}
		kasir, err := s.Rows(`SELECT e.full_name,e.employees_id,e.username,e.pincode,e.photo,e.level_id,m.level_name FROM m_employees e LEFT JOIN m_level m ON e.level_id=m.level_id WHERE e.level_id != ? AND e.passcode=?`, levelPrefix(employee["level_id"]), passcode)
		if err != nil {
			dbError(w, err)
			return
		}
		result := map[string]any{"data_employee": []map[string]any{employee}, "kasir": kasir, "status": "success"}
		if len(company) > 0 {
			result["company"] = company[0]
		} else {
			result["company"] = map[string]any{}
		}
		writeJSON(w, 200, result)
	}
}

func levelPrefix(value any) string {
	text, _ := value.(string)
	if len(text) > 12 {
		return text[:12] + "00001"
	}
	return text
}
