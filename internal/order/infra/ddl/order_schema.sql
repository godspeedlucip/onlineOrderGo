CREATE TABLE IF NOT EXISTS `orders` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `number` VARCHAR(64) NOT NULL,
  `user_id` BIGINT NOT NULL,
  `status` INT NOT NULL,
  `amount` BIGINT NOT NULL,
  `remark` VARCHAR(255) NOT NULL DEFAULT '',
  `version` BIGINT NOT NULL DEFAULT 1,
  `order_time` DATETIME(3) NOT NULL,
  `update_time` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_orders_number` (`number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `order_detail` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT NOT NULL,
  `item_type` VARCHAR(16) NOT NULL,
  `sku_id` BIGINT NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `flavor` VARCHAR(255) NOT NULL DEFAULT '',
  `quantity` BIGINT NOT NULL,
  `unit_amount` BIGINT NOT NULL,
  `line_amount` BIGINT NOT NULL,
  `create_time` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_order_detail_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `order_table_index` (
  `order_id` BIGINT NOT NULL,
  `table_name` VARCHAR(64) NOT NULL,
  `order_no` VARCHAR(64) NOT NULL,
  `created_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`order_id`),
  UNIQUE KEY `uk_order_table_index_no` (`order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `order_outbox` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(128) NOT NULL,
  `event_type` VARCHAR(64) NOT NULL,
  `order_id` BIGINT NOT NULL,
  `order_no` VARCHAR(64) NOT NULL,
  `payload` JSON NOT NULL,
  `status` VARCHAR(16) NOT NULL,
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME(3) NOT NULL,
  `created_at` DATETIME(3) NOT NULL,
  `updated_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_outbox_event` (`event_id`),
  KEY `idx_order_outbox_poll` (`status`, `next_retry_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
