CREATE TABLE deposits (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    currency        VARCHAR(10) NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    from_address    VARCHAR(128) NOT NULL,
    to_address      VARCHAR(128) NOT NULL,
    tx_hash         VARCHAR(128) NOT NULL UNIQUE,
    confirmations   INT DEFAULT 0,
    status          TINYINT DEFAULT 0,
    notified        TINYINT(1) DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at    DATETIME DEFAULT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_tx_hash (tx_hash),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
