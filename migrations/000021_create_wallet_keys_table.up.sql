CREATE TABLE IF NOT EXISTS wallet_keys (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    address VARCHAR(64) NOT NULL,
    encrypted_key TEXT NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX idx_wallet_keys_user_id (user_id),
    INDEX idx_wallet_keys_address (address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
