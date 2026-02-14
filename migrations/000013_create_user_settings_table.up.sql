CREATE TABLE user_settings (
    id                   BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id              BIGINT UNSIGNED NOT NULL UNIQUE,
    language             VARCHAR(10) DEFAULT 'zh-CN',
    small_free_usdt      DECIMAL(20,8) DEFAULT 100,
    small_free_trx       DECIMAL(20,8) DEFAULT 100,
    daily_spent_usdt     DECIMAL(20,8) DEFAULT 0,
    daily_spent_trx      DECIMAL(20,8) DEFAULT 0,
    daily_spent_date     DATE DEFAULT NULL,
    notification_enabled TINYINT(1) DEFAULT 1,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
