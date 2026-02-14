# Security Policy

## Disclaimer

This project is for **educational and research purposes only**. It is NOT recommended for production use with real funds without thorough security auditing.

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **DO NOT** open a public GitHub issue
2. Email: [Create a GitHub Security Advisory](https://github.com/TGlimmer/TG_walletbot/security/advisories/new)

## Security Best Practices

When deploying this bot:

- **Never** commit real credentials (`config.yaml`, `.env`, private keys)
- **Always** use environment variables for sensitive configuration
- **Rotate** Bot Token and JWT Secret regularly
- **Enable** PIN verification for all financial operations
- **Use** HTTPS for all API endpoints
- **Restrict** database access to localhost only
- **Monitor** transaction logs for anomalies
