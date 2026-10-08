package httpapi

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/memesan/api-react/internal/store"
)

func subMenu(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := requestValue(r, "passcode")
		if p == "" {
			writeJSON(w, 400, map[string]any{"status": "error", "msg": "passcode dibutuhkan"})
			return
		}
		switch requestValue(r, "act") {
		case "groups":
			subGroups(w, s, p)
		case "create":
			subCreateGroup(w, r, s, p)
		case "update":
			subUpdateGroup(w, r, s, p)
		case "delete":
			subDeleteGroup(w, r, s, p)
		case "options":
			subOptions(w, r, s, p)
		case "create_option":
			subCreateOption(w, r, s, p)
		case "update_option":
			subUpdateOption(w, r, s, p)
		case "delete_option":
			subDeleteOption(w, r, s, p)
		default:
			writeJSON(w, 400, map[string]any{"status": "error", "msg": "Action tidak valid"})
		}
	}
}
func subGroups(w http.ResponseWriter, s *store.Store, p string) {
	rows, e := s.Rows("SELECT id,name FROM sub_menu_groups WHERE passcode=? ORDER BY id ASC", p)
	if e != nil {
		dbError(w, e)
		return
	}
	out := make([]map[string]any, 0)
	for _, v := range rows {
		out = append(out, map[string]any{"id": sval(v["id"]), "name": v["name"]})
	}
	writeJSON(w, 200, out)
}
func subCreateGroup(w http.ResponseWriter, r *http.Request, s *store.Store, p string) {
	d := bodyMap(r)
	if !required(w, d["name"], "name dibutuhkan") {
		return
	}
	res, e := s.DB.Exec("INSERT INTO sub_menu_groups(passcode,name) VALUES(?,?)", p, d["name"])
	if e != nil {
		dbError(w, e)
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, 200, map[string]any{"status": "success", "data": map[string]any{"id": strconv.FormatInt(id, 10), "name": d["name"]}})
}
func subUpdateGroup(w http.ResponseWriter, r *http.Request, s *store.Store, p string) {
	id := r.URL.Query().Get("id")
	if !required(w, id, "id dibutuhkan") {
		return
	}
	d := bodyMap(r)
	if !required(w, d["name"], "name dibutuhkan") {
		return
	}
	res, e := s.DB.Exec("UPDATE sub_menu_groups SET name=? WHERE passcode=? AND id=?", d["name"], p, id)
	if e != nil {
		dbError(w, e)
		return
	}
	n, _ := res.RowsAffected()
	writeJSON(w, 200, map[string]string{"status": map[bool]string{true: "success", false: "error"}[n > 0], "msg": map[bool]string{true: "Berhasil update", false: "Gagal update"}[n > 0]})
}
func subDeleteGroup(w http.ResponseWriter, r *http.Request, s *store.Store, p string) {
	id := r.URL.Query().Get("id")
	if !required(w, id, "id dibutuhkan") {
		return
	}
	var x int
	e := s.DB.QueryRow("SELECT id FROM sub_menu_groups WHERE passcode=? AND id=?", p, id).Scan(&x)
	if e == sql.ErrNoRows {
		writeJSON(w, 404, map[string]string{"status": "error", "msg": "Grup sub menu tidak ditemukan"})
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	_, e = s.DB.Exec("DELETE FROM sub_menu_groups WHERE passcode=? AND id=?", p, id)
	if e != nil {
		dbError(w, e)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "success", "msg": "Berhasil hapus"})
}
func subOptions(w http.ResponseWriter, r *http.Request, s *store.Store, p string) {
	gid := r.URL.Query().Get("group_id")
	if !required(w, gid, "group_id dibutuhkan") {
		return
	}
	var x int
	e := s.DB.QueryRow("SELECT id FROM sub_menu_groups WHERE passcode=? AND id=?", p, gid).Scan(&x)
	if e == sql.ErrNoRows {
		writeJSON(w, 404, map[string]string{"status": "error", "msg": "Grup sub menu tidak ditemukan"})
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	rows, e := s.Rows("SELECT o.id,o.group_id,o.name,o.additional_price,o.sort_order,o.status_active FROM sub_menu_options o WHERE o.passcode=? AND o.group_id=? AND o.status_active='active' ORDER BY o.sort_order ASC,o.id ASC", p, gid)
	if e != nil {
		dbError(w, e)
		return
	}
	for _, v := range rows {
		v["id"] = sval(v["id"])
		v["group_id"] = sval(v["group_id"])
		v["additional_price"] = fval(v["additional_price"])
		v["sort_order"] = ival(v["sort_order"])
	}
	writeJSON(w, 200, map[string]any{"status": "success", "data": rows})
}
func subCreateOption(w http.ResponseWriter, r *http.Request, s *store.Store, p string) {
	d := bodyMap(r)
	if !required(w, d["group_id"], "group_id dibutuhkan") || !required(w, d["name"], "name dibutuhkan") {
		return
	}
	price := fval(d["additional_price"])
	if price < 0 {
		writeJSON(w, 400, map[string]string{"status": "error", "msg": "additional_price tidak boleh negatif"})
		return
	}
	sort := ival(d["sort_order"])
	var x int
	e := s.DB.QueryRow("SELECT id FROM sub_menu_groups WHERE passcode=? AND id=?", p, d["group_id"]).Scan(&x)
	if e == sql.ErrNoRows {
		writeJSON(w, 404, map[string]string{"status": "error", "msg": "Grup sub menu tidak ditemukan"})
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	res, e := s.DB.Exec("INSERT INTO sub_menu_options(passcode,group_id,name,additional_price,sort_order,status_active) VALUES(?,?,?,?,?,'active')", p, d["group_id"], d["name"], price, sort)
	if e != nil {
		dbError(w, e)
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, 200, map[string]any{"status": "success", "data": map[string]any{"id": strconv.FormatInt(id, 10), "group_id": sval(d["group_id"]), "name": d["name"], "additional_price": price, "sort_order": sort, "status_active": "active"}})
}
func subUpdateOption(w http.ResponseWriter, r *http.Request, s *store.Store, p string) {
	id := r.URL.Query().Get("id")
	if !required(w, id, "id dibutuhkan") {
		return
	}
	d := bodyMap(r)
	sets := []string{}
	args := []any{}
	if v, ok := d["name"]; ok {
		sets = append(sets, "name=?")
		args = append(args, v)
	}
	if v, ok := d["additional_price"]; ok {
		price := fval(v)
		if price < 0 {
			writeJSON(w, 400, map[string]string{"status": "error", "msg": "additional_price tidak boleh negatif"})
			return
		}
		sets = append(sets, "additional_price=?")
		args = append(args, price)
	}
	if v, ok := d["sort_order"]; ok {
		sets = append(sets, "sort_order=?")
		args = append(args, ival(v))
	}
	if len(sets) == 0 {
		writeJSON(w, 400, map[string]string{"status": "error", "msg": "Tidak ada field yang diupdate"})
		return
	}
	args = append(args, p, id)
	res, e := s.DB.Exec("UPDATE sub_menu_options SET "+strings.Join(sets, ",")+" WHERE passcode=? AND id=?", args...)
	if e != nil {
		dbError(w, e)
		return
	}
	n, _ := res.RowsAffected()
	writeJSON(w, 200, map[string]string{"status": map[bool]string{true: "success", false: "error"}[n > 0], "msg": map[bool]string{true: "Berhasil update", false: "Gagal update"}[n > 0]})
}
func subDeleteOption(w http.ResponseWriter, r *http.Request, s *store.Store, p string) {
	id := r.URL.Query().Get("id")
	if !required(w, id, "id dibutuhkan") {
		return
	}
	res, e := s.DB.Exec("UPDATE sub_menu_options SET status_active='unactive' WHERE passcode=? AND id=?", p, id)
	if e != nil {
		dbError(w, e)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, map[string]string{"status": "error", "msg": "Option tidak ditemukan"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "success", "msg": "Berhasil hapus"})
}
func sval(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	}
	return ""
}
func ival(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case []byte:
		n, _ := strconv.Atoi(string(x))
		return n
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}
func fval(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case []byte:
		n, _ := strconv.ParseFloat(string(x), 64)
		return n
	case string:
		n, _ := strconv.ParseFloat(x, 64)
		return n
	}
	return 0
}
