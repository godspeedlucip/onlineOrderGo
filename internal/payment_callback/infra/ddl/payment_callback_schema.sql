CREATE TABLE IF NOT EXISTS `payment_record` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT NOT NULL,
  `order_no` VARCHAR(64) NOT NULL,
  `transaction_no` VARCHAR(128) NOT NULL,
  `channel` VARCHAR(32) NOT NULL,
  `paid_amount` BIGINT NOT NULL,
  `paid_at` DATETIME(3) NOT NULL,
  `raw_status` VARCHAR(64) NOT NULL,
  `create_time` DATETIME(3) NOT NULL,
  `update_time` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_payment_record_txn` (`transaction_no`),
  KEY `idx_payment_record_order_no` (`order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `payment_callback_log` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `notify_id` VARCHAR(128) NOT NULL DEFAULT '',
  `order_no` VARCHAR(64) NOT NULL DEFAULT '',
  `transaction_no` VARCHAR(128) NOT NULL DEFAULT '',
  `channel` VARCHAR(32) NOT NULL DEFAULT '',
  `http_headers` JSON NULL,
  `body` LONGBLOB NULL,
  `verified` TINYINT(1) NOT NULL DEFAULT 0,
  `error_message` VARCHAR(512) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_callback_log_notify` (`notify_id`),
  KEY `idx_callback_log_order_no` (`order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `payment_outbox` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(192) NOT NULL,
  `event_type` VARCHAR(64) NOT NULL,
  `payload` JSON NOT NULL,
  `status` VARCHAR(16) NOT NULL,
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME(3) NOT NULL,
  `created_at` DATETIME(3) NOT NULL,
  `updated_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_payment_outbox_event` (`event_id`),
  KEY `idx_payment_outbox_poll` (`status`, `next_retry_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
