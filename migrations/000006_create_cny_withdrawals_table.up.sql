CREATE TABLE cny_withdrawals (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(20,2) NOT NULL,
    fee             DECIMAL(20,2) DEFAULT 0,
    method          VARCHAR(20) NOT NULL,
    account_name    VARCHAR(64) NOT NULL,
    account_no      VARCHAR(128) NOT NULL,
    bank_name       VARCHAR(64) DEFAULT '',
    status          TINYINT DEFAULT 0,
    reviewer_id     BIGINT UNSIGNED DEFAULT NULL,
    review_note     VARCHAR(255) DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at     DATETIME DEFAULT NULL,
    completed_at    DATETIME DEFAULT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
