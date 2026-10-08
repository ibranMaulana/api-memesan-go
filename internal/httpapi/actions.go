package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Every action is allow-listed. An absent or unknown act never falls through
// to a default mutation, which is important for payment and settlement APIs.
var endpointActions = map[string]map[string]bool{
	"/mob_absensi.php":         {"checkin": true, "checkLog": true, "checkout": true, "checkpoint": true, "delCheckpoint": true, "getProduct": true, "in": true, "kehadiran": true, "Km": true, "m": true, "Mi": true, "out": true, "statusAbsen": true, "tidakMasuk": true},
	"/mob_category.php":        {"delete": true, "list": true, "save": true, "update": true},
	"/mob_customer.php":        {"findCustomer": true, "saveCustomer": true},
	"/mob_dapur.php":           {"list": true, "list_by_items": true, "list_rekap": true, "ready": true, "ready_bulk": true, "reject": true, "reject_bulk": true, "summary": true, "summary_by_item": true, "test": true},
	"/mob_dapur_rekap.php":     {"list": true, "list_by_items": true, "ready": true, "ready_bulk": true, "summary": true},
	"/mob_draft_penjualan.php": {"delete": true, "deleteDraft": true, "detail": true, "getDraft": true, "list": true, "listDraft": true, "save": true, "saveDraft": true},
	"/mob_employees.php":       {"updateEmployee": true}, "/mob_gaji.php": {"value": true},
	"/mob_hutang.php":      {"detailHutangCustomer": true, "historyDetailHutangCustomer": true, "hutangCabang": true, "hutangCustomer": true, "receivePayment": true},
	"/mob_kartu_stock.php": {"dataCabang": true, "kartuStock": true}, "/mob_kas.php": {"delete": true, "list": true, "save": true},
	"/mob_login.php": {"getPincode": true, "login": true, "pincodeSave": true, "saveCompanyRegister": true, "saveUserRegister": true, "verifikasiSMS": true},
	"/mob_meja.php":  {"delete": true, "list": true, "save": true, "update": true}, "/mob_menu_old.php": {"delete": true, "list": true, "save": true, "update": true, "upload": true},
	"/mob_menu.php":    {"delete": true, "list": true, "save": true, "set_in_stock": true, "update": true, "upload": true},
	"/mob_payroll.php": {"getPayroll": true, "getPayrollDetail": true}, "/mob_pembayaran.php": {"cancel": true, "cekid": true, "list": true, "savePembayaran": true, "savePembayaranNonProduk": true},
	"/mob_pembelian.php": {"getSupplier": true, "submit": true}, "/mob_price_list.php": {"category": true, "getProduct": true, "searchProduct": true, "test": true, "uploadFotoProduct": true},
	"/mob_product.php": {"category": true, "getProductReady": true, "productDetail": true, "productSave": true, "productUpdate": true, "searchProduct": true, "test": true, "uploadFotoProduct": true},
	"/mob_promo.php":   {"data": true, "getData": true}, "/mob_request_moving.php": {"apply": true, "dataCabang": true, "dataMoving": true, "detailMoving": true, "receive": true, "request": true},
	"/mob_reservasi.php": {"cancel": true, "cekid": true, "list": true, "save": true}, "/mob_saldo.php": {"getSaldo": true, "saldoCabang": true, "saveSetor": true},
	"/mob_sales_omset.php": {"cekModal": true, "sales_omset": true}, "/mob_setor.php": {"acceptSetor": true, "save": true},
	"/mob_siswa.php": {"dataSiswa": true}, "/mob_stock_opname.php": {"saveStockOpname": true},
	"/mob_sub_menu.php": {"create": true, "create_option": true, "delete": true, "delete_option": true, "groups": true, "options": true, "update": true, "update_option": true},
	"/mob_supply.php":   {"apply": true, "receive": true, "request": true}, "/mob_umum.php": {"check_update": true, "get": true, "get_last_counters": true, "save": true}, "/mob_user.php": {"updatePin": true},
}

func guardedNotMigrated(endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := requestValue(r, "act")
		if actions, exists := endpointActions[endpoint]; exists {
			if action == "" {
				writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "message": "parameter act dibutuhkan"})
				return
			}
			if !actions[action] {
				writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "message": "act tidak dikenal", "act": action})
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]any{"status": "error", "message": "endpoint tidak ditemukan", "endpoint": endpoint, "act": action})
	}
}

func requestValue(r *http.Request, key string) string {
	if value := r.URL.Query().Get(key); value != "" {
		return value
	}
	if err := r.ParseForm(); err == nil {
		if value := r.FormValue(key); value != "" {
			return value
		}
	}
	if r.Body != nil && r.Header.Get("Content-Type") == "application/json" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return ""
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		var values map[string]any
		if json.Unmarshal(body, &values) == nil {
			if value, ok := values[key].(string); ok {
				return value
			}
		}
	}
	return ""
}
