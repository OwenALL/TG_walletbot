-- 商户表升级: 支持多商户自助创建模式
-- 1. 删除 user_id 唯一索引，改为普通索引 (允许一个用户创建多个商户)
-- 2. 添加 logo, fee_rate, ip_whitelist 字段

-- 删除旧的唯一索引
ALTER TABLE merchants DROP INDEX uk_user_id;

-- 添加普通索引
ALTER TABLE merchants ADD INDEX idx_user_id (user_id);

-- 添加新字段
ALTER TABLE merchants ADD COLUMN logo VARCHAR(255) NOT NULL DEFAULT '' COMMENT '商户LOGO文件ID' AFTER webhook_url;
ALTER TABLE merchants ADD COLUMN fee_rate DECIMAL(5,2) NOT NULL DEFAULT 0.00 COMMENT '费率百分比' AFTER logo;
ALTER TABLE merchants ADD COLUMN ip_whitelist TEXT COMMENT 'IP白名单(JSON数组)' AFTER fee_rate;
