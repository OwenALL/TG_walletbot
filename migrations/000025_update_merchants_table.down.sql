-- 回滚: 恢复唯一索引，删除新字段
ALTER TABLE merchants DROP COLUMN ip_whitelist;
ALTER TABLE merchants DROP COLUMN fee_rate;
ALTER TABLE merchants DROP COLUMN logo;

ALTER TABLE merchants DROP INDEX idx_user_id;
ALTER TABLE merchants ADD UNIQUE KEY uk_user_id (user_id);
