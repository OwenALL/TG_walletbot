CREATE TABLE exchange_rates (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    from_currency   VARCHAR(10) NOT NULL,
    to_currency     VARCHAR(10) NOT NULL,
    rate            DECIMAL(20,8) NOT NULL,
    spread          DECIMAL(10,4) DEFAULT 0,
    min_amount      DECIMAL(20,8) DEFAULT 0,
    max_amount      DECIMAL(20,8) DEFAULT 0,
    enabled         TINYINT(1) DEFAULT 1,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_pair (from_currency, to_currency)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
