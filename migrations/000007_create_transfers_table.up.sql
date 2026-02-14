CREATE TABLE transfers (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    from_user_id    BIGINT UNSIGNED NOT NULL,
    to_user_id      BIGINT UNSIGNED NOT NULL,
    currency        VARCHAR(10) NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    type            TINYINT DEFAULT 1,
    status          TINYINT DEFAULT 0,
    memo            VARCHAR(255) DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME DEFAULT NULL,
    INDEX idx_from_user (from_user_id),
    INDEX idx_to_user (to_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
