package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/memesan/api-react/internal/store"
)

type refundRequest struct {
	RefundID              string       `json:"refundId"`
	OriginalTransactionID string       `json:"originalTransactionId"`
	OriginalReceiptNumber string       `json:"originalReceiptNumber"`
	RefundedBy            string       `json:"refundedBy"`
	RefundedAt            string       `json:"refundedAt"`
	SubtotalRefund        float64      `json:"subtotalRefund"`
	DiscountRefund        float64      `json:"discountRefund"`
	ServiceRefund         float64      `json:"serviceRefund"`
	TaxRefund             float64      `json:"taxRefund"`
	TotalRefund           float64      `json:"totalRefund"`
	Items                 []refundItem `json:"itemsRefunded"`
}
type refundItem struct {
	ItemID       string  `json:"itemId"`
	ItemName     string  `json:"itemName"`
	RefundedQty  int     `json:"refundedQty"`
	RefundAmount float64 `json:"refundAmount"`
}

func refund(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && requestValue(r, "act") == "list" {
			refundList(w, r, s)
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]any{"status": "error", "error_message": "Method not allowed", "status_code": 405})
			return
		}
		var input refundRequest
		if err := decodeJSON(r, &input); err != nil {
			writeJSON(w, 400, map[string]string{"status": "error", "error_message": "Invalid JSON payload"})
			return
		}
		if input.RefundID == "" || input.OriginalTransactionID == "" || input.RefundedBy == "" {
			writeJSON(w, 400, map[string]any{"status": "error", "error_message": "Invalid refund payload", "status_code": 400})
			return
		}
		tx, err := s.DB.BeginTx(r.Context(), nil)
		if err != nil {
			dbError(w, err)
			return
		}
		defer tx.Rollback()
		resolved := input.OriginalTransactionID
		var exists string
		err = tx.QueryRowContext(r.Context(), "SELECT id_penjualan FROM t_penjualan WHERE id_penjualan=? LIMIT 1", resolved).Scan(&exists)
		if err == sql.ErrNoRows && input.OriginalReceiptNumber != "" {
			err = tx.QueryRowContext(r.Context(), "SELECT id_penjualan FROM t_penjualan WHERE receipt_number=? LIMIT 1", input.OriginalReceiptNumber).Scan(&resolved)
		}
		if err == sql.ErrNoRows {
			writeJSON(w, 404, map[string]any{"status": "error", "error_message": "Original transaction not found", "status_code": 404})
			return
		}
		if err != nil {
			dbError(w, err)
			return
		}
		var duplicate string
		if err = tx.QueryRowContext(r.Context(), "SELECT refund_id FROM refund_transaction WHERE refund_id=? LIMIT 1", input.RefundID).Scan(&duplicate); err == nil {
			writeJSON(w, 409, map[string]any{"status": "error", "error_message": "Refund ID already exists", "status_code": 409})
			return
		} else if err != sql.ErrNoRows {
			dbError(w, err)
			return
		}
		for _, item := range input.Items {
			var original int
			if err := tx.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(qty),0) FROM t_penjualan_detail WHERE id_penjualan=? AND id_item=?", resolved, item.ItemID).Scan(&original); err != nil {
				dbError(w, err)
				return
			}
			if original == 0 {
				writeJSON(w, 400, map[string]any{"status": "error", "error_message": "Item '" + item.ItemName + "' (ID: " + item.ItemID + ") not found in original transaction", "status_code": 400})
				return
			}
			var previous int
			if err := tx.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(ri.refunded_qty),0) FROM refund_transaction_items ri JOIN refund_transaction rt ON ri.refund_id=rt.refund_id WHERE rt.original_transaction_id=? AND ri.item_id=?", resolved, item.ItemID).Scan(&previous); err != nil {
				dbError(w, err)
				return
			}
			if item.RefundedQty+previous > original {
				writeJSON(w, 409, map[string]any{"status": "error", "error_message": "Item " + item.ItemName + " sudah direfund " + itoa(previous) + " dari " + itoa(original) + " qty. Sisa refund: " + itoa(original-previous), "status_code": 409})
				return
			}
		}
		refundedAt := time.Now().Format("2006-01-02 15:04:05")
		if input.RefundedAt != "" {
			if parsed, e := time.Parse(time.RFC3339, input.RefundedAt); e == nil {
				refundedAt = parsed.Format("2006-01-02 15:04:05")
			}
		}
		_, err = tx.ExecContext(r.Context(), `INSERT INTO refund_transaction (refund_id,original_transaction_id,refunded_by,refunded_at,subtotal_refund,discount_refund,service_refund,tax_refund,total_refund) VALUES (?,?,?,?,?,?,?,?,?)`, input.RefundID, resolved, input.RefundedBy, refundedAt, input.SubtotalRefund, input.DiscountRefund, input.ServiceRefund, input.TaxRefund, input.TotalRefund)
		if err != nil {
			dbError(w, err)
			return
		}
		totalRefunded := 0
		for _, item := range input.Items {
			var original int
			_ = tx.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(qty),0) FROM t_penjualan_detail WHERE id_penjualan=? AND id_item=?", resolved, item.ItemID).Scan(&original)
			_, err = tx.ExecContext(r.Context(), "INSERT INTO refund_transaction_items (refund_id,item_id,item_name,original_qty,refunded_qty,refund_amount) VALUES (?,?,?,?,?,?)", input.RefundID, item.ItemID, item.ItemName, original, item.RefundedQty, item.RefundAmount)
			if err != nil {
				dbError(w, err)
				return
			}
			totalRefunded += item.RefundedQty
		}
		var originalTotal, allRefunded int
		if err = tx.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(qty),0) FROM t_penjualan_detail WHERE id_penjualan=?", resolved).Scan(&originalTotal); err != nil {
			dbError(w, err)
			return
		}
		if err = tx.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(ri.refunded_qty),0) FROM refund_transaction_items ri JOIN refund_transaction rt ON ri.refund_id=rt.refund_id WHERE rt.original_transaction_id=?", resolved).Scan(&allRefunded); err != nil {
			dbError(w, err)
			return
		}
		_ = totalRefunded
		status := "none"
		if allRefunded >= originalTotal {
			status = "full"
		} else if allRefunded > 0 {
			status = "partial"
		}
		if _, err = tx.ExecContext(r.Context(), "UPDATE t_penjualan SET refund_status=? WHERE id_penjualan=?", status, resolved); err != nil {
			dbError(w, err)
			return
		}
		if err = tx.Commit(); err != nil {
			dbError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"status": "success", "data": map[string]string{"refundId": input.RefundID, "refundStatus": status}})
	}
}

func refundList(w http.ResponseWriter, r *http.Request, s *store.Store) {
	id := r.URL.Query().Get("id_transaksi")
	if id == "" {
		writeJSON(w, 400, map[string]any{"status": "error", "error_message": "Parameter id_transaksi is required", "status_code": 400})
		return
	}
	rows, e := s.Rows("SELECT rt.refund_id,rt.original_transaction_id,rt.refunded_by,COALESCE(e.full_name,'') refunded_by_name,rt.refunded_at,rt.subtotal_refund,rt.discount_refund,rt.service_refund,rt.tax_refund,rt.total_refund FROM refund_transaction rt LEFT JOIN m_employees e ON rt.refunded_by=e.employees_id WHERE rt.original_transaction_id=? ORDER BY rt.created_at ASC", id)
	if e != nil {
		dbError(w, e)
		return
	}
	for _, row := range rows {
		rid := row["refund_id"]
		items, e := s.Rows("SELECT item_id,item_name,refund_amount FROM refund_transaction_items WHERE refund_id=?", rid)
		if e != nil {
			dbError(w, e)
			return
		}
		row["itemsRefunded"] = items
	}
	writeJSON(w, 200, map[string]any{"status": "success", "data": rows})
}

func decodeJSON(r *http.Request, value any) error { return json.NewDecoder(r.Body).Decode(value) }
func itoa(v int) string                           { return strconv.Itoa(v) }
