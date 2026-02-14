CREATE TABLE admin_logs (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    admin_id        BIGINT UNSIGNED NOT NULL,
    action          VARCHAR(64) NOT NULL,
    target_type     VARCHAR(32) DEFAULT '',
    target_id       BIGINT UNSIGNED DEFAULT NULL,
    detail          JSON DEFAULT NULL,
    ip_address      VARCHAR(45) DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_admin_id (admin_id),
    INDEX idx_action (action),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
