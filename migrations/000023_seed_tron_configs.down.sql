-- 回滚: 删除 TRON 运营参数
DELETE FROM system_configs WHERE config_key IN (
    'tron_fee_limit',
    'tron_usdt_contract',
    'tron_grpc_timeout',
    'tron_grpc_batch_timeout'
);
