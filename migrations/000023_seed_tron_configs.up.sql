-- 将 TRON 运营参数从 config.yaml 迁移到 system_configs 表
-- 管理员可通过后台动态调整，无需重启服务
INSERT INTO system_configs (config_key, config_value, description) VALUES
('tron_fee_limit', '100000000', 'TRC20 交易手续费上限 (sun)，默认 100 TRX'),
('tron_usdt_contract', 'TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t', 'USDT TRC20 合约地址 (主网)'),
('tron_grpc_timeout', '10', 'gRPC 单次调用超时 (秒)'),
('tron_grpc_batch_timeout', '30', 'gRPC 批量调用超时 (秒)');
