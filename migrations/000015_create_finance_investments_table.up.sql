CREATE TABLE finance_investments (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    annual_rate     DECIMAL(10,4) NOT NULL,
    status          TINYINT DEFAULT 1,
    interest_start  DATETIME NOT NULL,
    total_profit    DECIMAL(20,8) DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    withdrawn_at    DATETIME DEFAULT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
