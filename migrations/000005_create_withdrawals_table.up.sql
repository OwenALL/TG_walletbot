CREATE TABLE withdrawals (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    currency        VARCHAR(10) NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    fee             DECIMAL(20,8) NOT NULL,
    actual_amount   DECIMAL(20,8) NOT NULL,
    to_address      VARCHAR(128) NOT NULL,
    tx_hash         VARCHAR(128) DEFAULT '',
    status          TINYINT DEFAULT 0,
    reviewer_id     BIGINT UNSIGNED DEFAULT NULL,
    review_note     VARCHAR(255) DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at     DATETIME DEFAULT NULL,
    completed_at    DATETIME DEFAULT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
