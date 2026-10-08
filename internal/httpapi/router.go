package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/memesan/api-react/internal/config"
	"github.com/memesan/api-react/internal/store"
)

func New(s *store.Store, _ config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if err := s.DB.Ping(); err != nil {
			writeJSON(w, 503, map[string]string{"status": "unready"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/mob_bank.php", bank(s))
	mux.HandleFunc("/mob_notes.php", notes(s))
	mux.HandleFunc("/mob_items.php", items(s))
	mux.HandleFunc("/mob_dashboard.php", dashboard(s))
	mux.HandleFunc("/mob_rekap_list.php", recapList(s))
	mux.HandleFunc("/mob_login.php", login(s))
	mux.HandleFunc("/mob_category.php", category(s))
	mux.HandleFunc("/mob_meja.php", meja(s))
	mux.HandleFunc("/mob_kas.php", kas(s))
	mux.HandleFunc("/mob_customer.php", customer(s))
	mux.HandleFunc("/mob_user.php", user(s))
	mux.HandleFunc("/mob_refund.php", refund(s))
	mux.HandleFunc("/mob_sub_menu.php", subMenu(s))
	mux.HandleFunc("/mob_umum.php", umum(s))
	mux.HandleFunc("/mob_promo.php", promo(s))
	mux.HandleFunc("/mob_product.php", product(s))
	mux.HandleFunc("/mob_dapur.php", dapur(s))
	mux.HandleFunc("/mob_menu.php", menu(s))
	mux.HandleFunc("/mob_reservasi.php", reservasi(s))
	mux.HandleFunc("/mob_payroll.php", payroll(s))
	mux.HandleFunc("/mob_siswa.php", siswa(s))
	mux.HandleFunc("/mob_employees.php", employees(s))
	mux.HandleFunc("/mob_sales_omset.php", salesOmset(s))
	mux.HandleFunc("/mob_stock_opname.php", stockOpname(s))
	mux.HandleFunc("/mob_saldo.php", saldo(s))
	mux.HandleFunc("/mob_setor.php", setor(s))
	mux.HandleFunc("/table-draft/snapshots.php", snapshots(s))
	mux.HandleFunc("/mob_hutang.php", hutang(s))
	mux.HandleFunc("/mob_kartu_stock.php", kartuStock(s))
	mux.HandleFunc("/mob_price_list.php", priceList(s))
	mux.HandleFunc("/mob_pembelian.php", pembelian(s))
	mux.HandleFunc("/mob_absensi.php", absensi(s))
	mux.HandleFunc("/mob_request_moving.php", requestMoving(s))
	mux.HandleFunc("/mob_dapur_rekap.php", dapurRekap(s))
	mux.HandleFunc("/mob_pembayaran.php", pembayaran(s))
	mux.HandleFunc("/mob_gaji.php", gaji(s))
	mux.HandleFunc("/mob_menu_old.php", menu(s))
	mux.HandleFunc("/mob_supply.php", supply(s))
	mux.HandleFunc("/mob_draft_penjualan.php", draftPenjualan(s))
	mux.HandleFunc("/mob_rekap.php", rekap(s))
	mux.HandleFunc("/mob_rekap_update.php", rekap(s))
	for _, path := range endpointPaths {
		if !nativePath(path) {
			mux.HandleFunc(path, guardedNotMigrated(path))
		}
	}
	for _, path := range endpointPaths {
		cleanPath := strings.TrimSuffix(path, ".php")
		switch path {
		case "/mob_bank.php":
			mux.HandleFunc(cleanPath, bank(s))
		case "/mob_notes.php":
			mux.HandleFunc(cleanPath, notes(s))
		case "/mob_items.php":
			mux.HandleFunc(cleanPath, items(s))
		case "/mob_dashboard.php":
			mux.HandleFunc(cleanPath, dashboard(s))
		case "/mob_rekap_list.php":
			mux.HandleFunc(cleanPath, recapList(s))
		case "/mob_login.php":
			mux.HandleFunc(cleanPath, login(s))
		case "/mob_category.php":
			mux.HandleFunc(cleanPath, category(s))
		case "/mob_meja.php":
			mux.HandleFunc(cleanPath, meja(s))
		case "/mob_kas.php":
			mux.HandleFunc(cleanPath, kas(s))
		case "/mob_customer.php":
			mux.HandleFunc(cleanPath, customer(s))
		case "/mob_user.php":
			mux.HandleFunc(cleanPath, user(s))
		case "/mob_refund.php":
			mux.HandleFunc(cleanPath, refund(s))
		case "/mob_sub_menu.php":
			mux.HandleFunc(cleanPath, subMenu(s))
		case "/mob_umum.php":
			mux.HandleFunc(cleanPath, umum(s))
		case "/mob_promo.php":
			mux.HandleFunc(cleanPath, promo(s))
		case "/mob_product.php":
			mux.HandleFunc(cleanPath, product(s))
		case "/mob_dapur.php":
			mux.HandleFunc(cleanPath, dapur(s))
		case "/mob_menu.php":
			mux.HandleFunc(cleanPath, menu(s))
		case "/mob_reservasi.php":
			mux.HandleFunc(cleanPath, reservasi(s))
		case "/mob_payroll.php":
			mux.HandleFunc(cleanPath, payroll(s))
		case "/mob_siswa.php":
			mux.HandleFunc(cleanPath, siswa(s))
		case "/mob_employees.php":
			mux.HandleFunc(cleanPath, employees(s))
		case "/mob_sales_omset.php":
			mux.HandleFunc(cleanPath, salesOmset(s))
		case "/mob_stock_opname.php":
			mux.HandleFunc(cleanPath, stockOpname(s))
		case "/mob_saldo.php":
			mux.HandleFunc(cleanPath, saldo(s))
		case "/mob_setor.php":
			mux.HandleFunc(cleanPath, setor(s))
		case "/table-draft/snapshots.php":
			mux.HandleFunc(cleanPath, snapshots(s))
		case "/mob_hutang.php":
			mux.HandleFunc(cleanPath, hutang(s))
		case "/mob_kartu_stock.php":
			mux.HandleFunc(cleanPath, kartuStock(s))
		case "/mob_price_list.php":
			mux.HandleFunc(cleanPath, priceList(s))
		case "/mob_pembelian.php":
			mux.HandleFunc(cleanPath, pembelian(s))
		case "/mob_absensi.php":
			mux.HandleFunc(cleanPath, absensi(s))
		case "/mob_request_moving.php":
			mux.HandleFunc(cleanPath, requestMoving(s))
		case "/mob_dapur_rekap.php":
			mux.HandleFunc(cleanPath, dapurRekap(s))
		case "/mob_pembayaran.php":
			mux.HandleFunc(cleanPath, pembayaran(s))
		case "/mob_gaji.php":
			mux.HandleFunc(cleanPath, gaji(s))
		case "/mob_menu_old.php":
			mux.HandleFunc(cleanPath, menu(s))
		case "/mob_supply.php":
			mux.HandleFunc(cleanPath, supply(s))
		case "/mob_draft_penjualan.php":
			mux.HandleFunc(cleanPath, draftPenjualan(s))
		case "/mob_rekap.php", "/mob_rekap_update.php":
			mux.HandleFunc(cleanPath, rekap(s))
		default:
			mux.HandleFunc(cleanPath, guardedNotMigrated(path))
		}
	}
	// Canonical action URLs: /mob_product/productSave. The query-string form
	// remains supported above for Android compatibility.
	for endpoint, actions := range endpointActions {
		base := strings.TrimSuffix(endpoint, ".php")
		for action := range actions {
			endpoint, action := endpoint, action
			path := base + "/" + action
			switch endpoint {
			case "/mob_login.php":
				if action == "login" {
					mux.HandleFunc(path, login(s))
					continue
				}
			case "/mob_refund.php":
				if action == "list" {
					mux.HandleFunc(path, refund(s))
					continue
				}
			case "/mob_bank.php":
				mux.HandleFunc(path, bank(s))
				continue
			case "/mob_notes.php":
				mux.HandleFunc(path, notes(s))
				continue
			case "/mob_items.php":
				mux.HandleFunc(path, items(s))
				continue
			case "/mob_dashboard.php":
				mux.HandleFunc(path, dashboard(s))
				continue
			case "/mob_rekap_list.php":
				mux.HandleFunc(path, recapList(s))
				continue
			case "/mob_category.php":
				mux.HandleFunc(path, category(s))
				continue
			case "/mob_meja.php":
				mux.HandleFunc(path, meja(s))
				continue
			case "/mob_kas.php":
				mux.HandleFunc(path, kas(s))
				continue
			case "/mob_customer.php":
				mux.HandleFunc(path, customer(s))
				continue
			case "/mob_user.php":
				mux.HandleFunc(path, user(s))
				continue
			case "/mob_sub_menu.php":
				mux.HandleFunc(path, subMenu(s))
				continue
			case "/mob_umum.php":
				mux.HandleFunc(path, umum(s))
				continue
			case "/mob_promo.php":
				mux.HandleFunc(path, promo(s))
				continue
			case "/mob_product.php":
				mux.HandleFunc(path, product(s))
				continue
			case "/mob_dapur.php":
				mux.HandleFunc(path, dapur(s))
				continue
			case "/mob_menu.php":
				mux.HandleFunc(path, menu(s))
				continue
			case "/mob_reservasi.php":
				mux.HandleFunc(path, reservasi(s))
				continue
			case "/mob_payroll.php":
				mux.HandleFunc(path, payroll(s))
				continue
			case "/mob_siswa.php":
				mux.HandleFunc(path, siswa(s))
				continue
			case "/mob_employees.php":
				mux.HandleFunc(path, employees(s))
				continue
			case "/mob_sales_omset.php":
				mux.HandleFunc(path, salesOmset(s))
				continue
			case "/mob_stock_opname.php":
				mux.HandleFunc(path, stockOpname(s))
				continue
			case "/mob_saldo.php":
				mux.HandleFunc(path, saldo(s))
				continue
			case "/mob_setor.php":
				mux.HandleFunc(path, setor(s))
				continue
			case "/mob_hutang.php":
				mux.HandleFunc(path, hutang(s))
				continue
			case "/mob_kartu_stock.php":
				mux.HandleFunc(path, kartuStock(s))
				continue
			case "/mob_price_list.php":
				mux.HandleFunc(path, priceList(s))
				continue
			case "/mob_pembelian.php":
				mux.HandleFunc(path, pembelian(s))
				continue
			case "/mob_absensi.php":
				mux.HandleFunc(path, absensi(s))
				continue
			case "/mob_request_moving.php":
				mux.HandleFunc(path, requestMoving(s))
				continue
			case "/mob_dapur_rekap.php":
				mux.HandleFunc(path, dapurRekap(s))
				continue
			case "/mob_pembayaran.php":
				mux.HandleFunc(path, pembayaran(s))
				continue
			case "/mob_gaji.php":
				mux.HandleFunc(path, gaji(s))
				continue
			case "/mob_menu_old.php":
				mux.HandleFunc(path, menu(s))
				continue
			case "/mob_supply.php":
				mux.HandleFunc(path, supply(s))
				continue
			case "/mob_draft_penjualan.php":
				mux.HandleFunc(path, draftPenjualan(s))
				continue
			case "/mob_rekap.php", "/mob_rekap_update.php":
				mux.HandleFunc(path, rekap(s))
				continue
			}
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusNotFound, map[string]any{"status": "error", "message": "endpoint tidak ditemukan", "endpoint": endpoint, "act": action})
			})
		}
	}
	return accessLog(mux)
}

func nativePath(path string) bool {
	return path == "/mob_bank.php" || path == "/mob_notes.php" || path == "/mob_items.php" || path == "/mob_dashboard.php" || path == "/mob_rekap_list.php" || path == "/mob_login.php" || path == "/mob_category.php" || path == "/mob_meja.php" || path == "/mob_kas.php" || path == "/mob_customer.php" || path == "/mob_user.php" || path == "/mob_refund.php" || path == "/mob_sub_menu.php" || path == "/mob_umum.php" || path == "/mob_promo.php" || path == "/mob_product.php" || path == "/mob_dapur.php" || path == "/mob_menu.php" || path == "/mob_reservasi.php" || path == "/mob_payroll.php" || path == "/mob_siswa.php" || path == "/mob_employees.php" || path == "/mob_sales_omset.php" || path == "/mob_stock_opname.php" || path == "/mob_saldo.php" || path == "/mob_setor.php" || path == "/table-draft/snapshots.php" || path == "/mob_hutang.php" || path == "/mob_kartu_stock.php" || path == "/mob_price_list.php" || path == "/mob_pembelian.php" || path == "/mob_absensi.php" || path == "/mob_request_moving.php" || path == "/mob_dapur_rekap.php" || path == "/mob_pembayaran.php" || path == "/mob_gaji.php" || path == "/mob_menu_old.php" || path == "/mob_supply.php" || path == "/mob_draft_penjualan.php" || path == "/mob_rekap.php" || path == "/mob_rekap_update.php"
}

var endpointPaths = []string{"/mob_absensi.php", "/mob_bank.php", "/mob_category.php", "/mob_customer.php", "/mob_dapur.php", "/mob_dapur_rekap.php", "/mob_dashboard.php", "/mob_draft_penjualan.php", "/mob_employees.php", "/mob_gaji.php", "/mob_hutang.php", "/mob_items.php", "/mob_kartu_stock.php", "/mob_kas.php", "/mob_login.php", "/mob_meja.php", "/mob_menu_old.php", "/mob_menu.php", "/mob_notes.php", "/mob_payroll.php", "/mob_pembayaran.php", "/mob_pembelian.php", "/mob_price_list.php", "/mob_product.php", "/mob_promo.php", "/mob_refund.php", "/mob_rekap_list.php", "/mob_rekap_old.php", "/mob_rekap.php", "/mob_rekap_update.php", "/mob_request_moving.php", "/mob_reservasi.php", "/mob_saldo.php", "/mob_sales_omset.php", "/mob_setor.php", "/mob_siswa.php", "/mob_stock_opname.php", "/mob_sub_menu.php", "/mob_supply.php", "/mob_umum.php", "/mob_user.php", "/table-draft/snapshots.php"}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
}
