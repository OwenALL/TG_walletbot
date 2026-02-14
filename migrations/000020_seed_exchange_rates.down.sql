DELETE FROM exchange_rates WHERE (from_currency, to_currency) IN (
    ('USDT', 'CNY'),
    ('CNY',  'USDT'),
    ('USDT', 'TRX'),
    ('TRX',  'USDT'),
    ('TRX',  'CNY'),
    ('CNY',  'TRX')
);
