CREATE TABLE red_packets (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id         BIGINT UNSIGNED NOT NULL,
    currency        VARCHAR(10) NOT NULL,
    total_amount    DECIMAL(20,8) NOT NULL,
    total_count     INT NOT NULL,
    claimed_count   INT DEFAULT 0,
    claimed_amount  DECIMAL(20,8) DEFAULT 0,
    type            TINYINT DEFAULT 1,
    cover_file_id   VARCHAR(255) DEFAULT '',
    status          TINYINT DEFAULT 0,
    message_id      BIGINT DEFAULT NULL,
    chat_id         BIGINT DEFAULT NULL,
    expire_at       DATETIME NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_expire_at (expire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
