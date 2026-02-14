CREATE TABLE IF NOT EXISTS merchants (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    business_name VARCHAR(128) NOT NULL DEFAULT '',
    api_key VARCHAR(64) NOT NULL DEFAULT '',
    api_secret VARCHAR(128) NOT NULL DEFAULT '',
    webhook_url VARCHAR(255) NOT NULL DEFAULT '',
    status TINYINT DEFAULT 0 COMMENT '0:待审核 1:已通过 2:已拒绝 3:已禁用',
    description VARCHAR(512) DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE KEY uk_user_id (user_id),
    UNIQUE KEY uk_api_key (api_key),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
