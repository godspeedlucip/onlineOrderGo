-- Cart write model (aligned with cart repo implementation).
CREATE TABLE IF NOT EXISTS `cart` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  `item_type` VARCHAR(16) NOT NULL,
  `item_id` BIGINT NOT NULL,
  `flavor` VARCHAR(255) NOT NULL DEFAULT '',
  `name` VARCHAR(255) NOT NULL DEFAULT '',
  `image` VARCHAR(1024) NOT NULL DEFAULT '',
  `unit_price` BIGINT NOT NULL,
  `quantity` INT NOT NULL,
  `amount` BIGINT NOT NULL,
  `version` BIGINT NOT NULL DEFAULT 1,
  `create_time` DATETIME(3) NOT NULL,
  `update_time` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cart_user_item_flavor` (`user_id`, `item_type`, `item_id`, `flavor`),
  KEY `idx_cart_user_update_time` (`user_id`, `update_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Idempotency store for replaying success results.
CREATE TABLE IF NOT EXISTS `cart_idempotency` (
  `scene` VARCHAR(64) NOT NULL,
  `idem_key` VARCHAR(128) NOT NULL,
  `token` VARCHAR(64) NOT NULL,
  `status` VARCHAR(16) NOT NULL,
  `result_blob` LONGBLOB NULL,
  `reason` VARCHAR(512) NOT NULL DEFAULT '',
  `updated_at` DATETIME(3) NOT NULL,
  `expire_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`scene`, `idem_key`),
  KEY `idx_cart_idem_expire` (`expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
