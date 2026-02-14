CREATE TABLE transactions (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    type            VARCHAR(20) NOT NULL,
    currency        VARCHAR(10) NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    fee             DECIMAL(20,8) DEFAULT 0,
    balance_before  DECIMAL(20,8) NOT NULL,
    balance_after   DECIMAL(20,8) NOT NULL,
    related_id      BIGINT UNSIGNED DEFAULT NULL,
    related_type    VARCHAR(20) DEFAULT '',
    memo            VARCHAR(255) DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_type (type),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
