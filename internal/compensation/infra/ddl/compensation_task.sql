CREATE TABLE IF NOT EXISTS `compensation_task` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `task_id` VARCHAR(128) NOT NULL,
  `job_type` VARCHAR(64) NOT NULL,
  `biz_key` VARCHAR(255) NOT NULL DEFAULT '',
  `payload` JSON NULL,
  `status` VARCHAR(16) NOT NULL,
  `retry_count` INT NOT NULL DEFAULT 0,
  `max_retry` INT NOT NULL DEFAULT 3,
  `next_execute_at` DATETIME(3) NOT NULL,
  `scheduled_at` DATETIME(3) NOT NULL,
  `last_error` VARCHAR(512) NOT NULL DEFAULT '',
  `dead_reason` VARCHAR(512) NOT NULL DEFAULT '',
  `dead_at` DATETIME(3) NULL,
  `shard_table` VARCHAR(64) NOT NULL DEFAULT '',
  `lock_token` VARCHAR(128) NOT NULL DEFAULT '',
  `lock_expire_at` DATETIME(3) NULL,
  `created_at` DATETIME(3) NOT NULL,
  `updated_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_compensation_task_id` (`task_id`),
  KEY `idx_compensation_task_poll` (`job_type`, `status`, `next_execute_at`),
  KEY `idx_compensation_task_shard` (`shard_table`, `status`, `next_execute_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `compensation_task_run` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `task_id` VARCHAR(128) NOT NULL,
  `job_type` VARCHAR(64) NOT NULL,
  `status` VARCHAR(16) NOT NULL,
  `reason` VARCHAR(512) NOT NULL DEFAULT '',
  `retry_count` INT NOT NULL DEFAULT 0,
  `started_at` DATETIME(3) NOT NULL,
  `finished_at` DATETIME(3) NOT NULL,
  `duration_ms` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_compensation_run_task` (`task_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `compensation_task_dlq` (
  `task_id` VARCHAR(128) NOT NULL,
  `reason` VARCHAR(512) NOT NULL DEFAULT '',
  `archived_at` DATETIME(3) NOT NULL,
  PRIMARY KEY (`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
