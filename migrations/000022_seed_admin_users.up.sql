INSERT INTO admin_users (username, password_hash, role, status, created_at, updated_at)
VALUES ('admin', '$2a$10$H1G08Ra8bCzSMcY0VQE3eeh.G.iuD0E5FlfnP0B2XTQisxUKbAM0q', 'super_admin', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE role = VALUES(role), status = VALUES(status), updated_at = NOW();
