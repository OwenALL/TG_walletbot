CREATE TABLE wallets (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    currency        VARCHAR(10) NOT NULL,
    balance         DECIMAL(20,8) NOT NULL DEFAULT 0,
    frozen_balance  DECIMAL(20,8) NOT NULL DEFAULT 0,
    deposit_address VARCHAR(128) DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_currency (user_id, currency),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
