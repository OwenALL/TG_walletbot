CREATE TABLE exchanges (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    from_currency   VARCHAR(10) NOT NULL,
    to_currency     VARCHAR(10) NOT NULL,
    from_amount     DECIMAL(20,8) NOT NULL,
    to_amount       DECIMAL(20,8) NOT NULL,
    rate            DECIMAL(20,8) NOT NULL,
    fee             DECIMAL(20,8) DEFAULT 0,
    status          TINYINT DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
