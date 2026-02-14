CREATE TABLE sub_accounts (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    master_user_id  BIGINT UNSIGNED NOT NULL,
    sub_user_id     BIGINT UNSIGNED NOT NULL,
    status          TINYINT DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at    DATETIME DEFAULT NULL,
    UNIQUE KEY uk_master_sub (master_user_id, sub_user_id),
    INDEX idx_master (master_user_id),
    INDEX idx_sub (sub_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
