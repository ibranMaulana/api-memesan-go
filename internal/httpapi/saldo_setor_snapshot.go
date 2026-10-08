package httpapi

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/memesan/api-react/internal/store"
)

func setor(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
			return
		}
		switch requestValue(r, "act") {
		case "save":
			_, e := s.DB.Exec("INSERT INTO t_setor(id_setor,id_setor_comp,passcode,nominal_setor,id_branch,employees_id,tanggal) VALUES(?,?,?,?,?,?,CURRENT_DATE())", newID("STR"), newID("STR"), r.FormValue("passcode"), r.FormValue("nominal_setor"), r.FormValue("id_branch"), r.FormValue("employees_id"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
		case "acceptSetor":
			_, e := s.DB.Exec("UPDATE t_setor SET employees_id_accept=?,tgl_accept=CURRENT_DATE() WHERE id_setor=? AND passcode=?", r.URL.Query().Get("employees_id"), r.URL.Query().Get("id_setor"), r.FormValue("passcode"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
		default:
			writeJSON(w, 400, map[string]string{"msg": "act tidak dikenal"})
		}
	}
}
func saldo(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		act := requestValue(r, "act")
		if act == "saveSetor" {
			if err := r.ParseForm(); err != nil {
				writeJSON(w, 400, map[string]string{"msg": "request tidak valid"})
				return
			}
			_, e := s.DB.Exec("INSERT INTO t_setor(id_setor,id_setor_comp,passcode,nominal_setor,id_branch,employees_id,tanggal) VALUES(?,?,?,?,?,?,CURRENT_DATE())", newID("STR"), newID("STR"), r.FormValue("passcode"), r.FormValue("nominal_setor"), r.FormValue("id_branch"), r.FormValue("employees_id"))
			if e != nil {
				dbError(w, e)
				return
			}
			writeJSON(w, 200, "success")
			return
		}
		if act == "saldoCabang" {
			p := r.URL.Query().Get("passcode")
			rows, e := s.Rows("SELECT id_branch,branch_name FROM m_branch WHERE passcode=? AND status_active='active'", p)
			if e != nil {
				dbError(w, e)
				return
			}
			for _, row := range rows {
				var sales, deposit float64
				_ = s.DB.QueryRow("SELECT COALESCE(SUM(dp+paid_off),0) FROM t_penjualan WHERE passcode=? AND id_branch=?", p, row["id_branch"]).Scan(&sales)
				_ = s.DB.QueryRow("SELECT COALESCE(SUM(nominal_setor),0) FROM t_setor WHERE passcode=? AND id_branch=?", p, row["id_branch"]).Scan(&deposit)
				row["saldo"] = sales - deposit
			}
			writeJSON(w, 200, map[string]any{"data": rows})
			return
		}
		if act == "getSaldo" {
			if err := r.ParseForm(); err != nil {
				writeJSON(w, 400, "request tidak valid")
				return
			}
			p := r.FormValue("passcode")
			branch := r.FormValue("id_branch")
			start, end := mysqlDate(r.FormValue("tgl1")), mysqlDate(r.FormValue("tgl2"))
			before, _ := s.Rows("SELECT COALESCE(SUM(dp+paid_off),0) debit,0 kredit FROM t_penjualan WHERE passcode=? AND id_branch=? AND tgl_penjualan<? UNION ALL SELECT 0 debit,COALESCE(SUM(nominal_setor),0) kredit FROM t_setor WHERE passcode=? AND id_branch=? AND tanggal<?", p, branch, start, p, branch, start)
			rows, _ := s.Rows("SELECT 'A' typ,nomor_transaksi id,COALESCE(dp+paid_off,0) debit,0 kredit,tgl_penjualan date FROM t_penjualan WHERE passcode=? AND id_branch=? AND tgl_penjualan BETWEEN ? AND ? UNION ALL SELECT 'B' typ,id_setor id,0 debit,nominal_setor kredit,tanggal date FROM t_setor WHERE passcode=? AND id_branch=? AND tanggal BETWEEN ? AND ? ORDER BY date,typ", p, branch, start, end, p, branch, start, end)
			writeJSON(w, 200, map[string]any{"before": before, "now": rows, "after": []any{}, "tot_debit": 0, "tot_kredit": 0, "saldo": 0})
			return
		}
		invalidAction(w)
	}
}
func snapshots(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, 405, map[string]any{"status": false, "code": "METHOD_NOT_ALLOWED", "message": "Only POST method is allowed", "failed": []string{}})
			return
		}
		var input struct {
			Passcode string           `json:"passcode"`
			Events   []map[string]any `json:"events"`
		}
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			writeJSON(w, 400, map[string]any{"status": false, "code": "INVALID_JSON", "message": "Invalid JSON payload", "failed": []string{}})
			return
		}
		if input.Passcode == "" {
			writeJSON(w, 422, map[string]any{"status": false, "code": "VALIDATION_ERROR", "message": "passcode is required", "failed": []string{}})
			return
		}
		if len(input.Events) == 0 || len(input.Events) > 100 {
			writeJSON(w, 422, map[string]any{"status": false, "code": "VALIDATION_ERROR", "message": "events must be a non-empty array", "failed": []string{}})
			return
		}
		tx, e := s.DB.Begin()
		if e != nil {
			dbError(w, e)
			return
		}
		defer tx.Rollback()
		accepted, duplicates, failed := []string{}, []string{}, []string{}
		for i, event := range input.Events {
			id := fmt.Sprint(event["eventId"])
			if id == "<nil>" || id == "" {
				failed = append(failed, fmt.Sprintf("event_%d", i))
				continue
			}
			typ := fmt.Sprint(event["type"])
			if typ != "SAVE_SNAPSHOT" {
				failed = append(failed, id)
				continue
			}
			payload, _ := json.Marshal(event["transaction"])
			sum := md5.Sum(payload)
			hash := hex.EncodeToString(sum[:])
			var existing string
			err := tx.QueryRow("SELECT MD5(data_json) FROM draft_transaction_snapshots WHERE event_id=?", id).Scan(&existing)
			if err == nil {
				if existing == hash {
					duplicates = append(duplicates, id)
				} else {
					tx.Rollback()
					writeJSON(w, 409, map[string]any{"status": false, "code": "EVENT_ID_PAYLOAD_CONFLICT", "message": "eventId already exists with a different payload", "failed": []string{id}})
					return
				}
				continue
			}
			_, e = tx.Exec("INSERT INTO draft_transaction_snapshots(event_id,passcode,transaction_id,receipt_number,table_name,item_count,data_json,event_type,occurred_at,operator_id,operator_name,device_id,received_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)", id, input.Passcode, fmt.Sprint(event["transactionId"]), event["receiptNumber"], event["table"], ival(event["itemCount"]), string(payload), typ, event["occurredAt"], event["operatorId"], event["operatorName"], event["deviceId"], time.Now())
			if e != nil {
				dbError(w, e)
				return
			}
			accepted = append(accepted, id)
		}
		if e = tx.Commit(); e != nil {
			dbError(w, e)
			return
		}
		code := "OK"
		if len(duplicates) > 0 || len(failed) > 0 {
			code = "OK_WITH_DUPLICATES"
		}
		writeJSON(w, 200, map[string]any{"status": true, "code": code, "accepted": accepted, "duplicates": duplicates, "failed": failed})
	}
}
func newID(prefix string) string { return prefix + time.Now().Format("20060102150405.000000") }
