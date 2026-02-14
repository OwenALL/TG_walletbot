-- 红包新增功能字段: 备注、验证码、会员专属、隐私
ALTER TABLE red_packets
    ADD COLUMN remark VARCHAR(200) NOT NULL DEFAULT '' COMMENT '红包备注' AFTER cover_file_id,
    ADD COLUMN is_captcha TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否验证码红包' AFTER remark,
    ADD COLUMN is_premium_only TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否会员专属红包' AFTER is_captcha,
    ADD COLUMN is_private TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否隐私红包' AFTER is_premium_only;
