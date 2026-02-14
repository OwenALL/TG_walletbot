CREATE TABLE red_packet_claims (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    red_packet_id   BIGINT UNSIGNED NOT NULL,
    user_id         BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_packet_user (red_packet_id, user_id),
    INDEX idx_red_packet_id (red_packet_id),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
