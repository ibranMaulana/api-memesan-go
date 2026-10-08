-- Run after a production backup. Existing rows remain readable.
ALTER TABLE t_penjualan_detail
  ADD COLUMN sub_menu_id INT NULL AFTER id_item,
  ADD COLUMN sub_menu_name VARCHAR(255) NULL AFTER sub_menu_id,
  ADD COLUMN sub_menu_price_delta DECIMAL(15,2) NOT NULL DEFAULT 0 AFTER sub_menu_name;

ALTER TABLE t_penjualan_detail
  ADD KEY idx_detail_sub_menu (passcode, sub_menu_id);
