# Security Policy

## Reporting a Vulnerability

This policy covers the personal downstream fork at `clabs-public-forks/aerion`. No dedicated fork security email or response-time commitment is currently documented.

Do not post vulnerability details, credentials, or private mail data in public issues. If this fork's GitHub Security tab offers **Report a vulnerability**, use that private channel. Otherwise, arrange a private reporting channel with the fork owner before sharing sensitive details. Private vulnerability reporting availability has not been confirmed.

Include the affected fork commit, impact, reproduction steps, and a minimal proof of concept with sensitive data removed. State whether the issue also reproduces on upstream Aerion, if known.

For vulnerabilities affecting upstream Aerion, follow [upstream's security policy](https://github.com/hkdb/aerion/security/policy). Upstream's contact address and response commitments apply to upstream; they are not the reporting policy for fork-specific changes.

## Security Best Practices for Users

### OAuth Credentials

If you are compiling Aerion from source:

1. **Never commit OAuth credentials** to version control
2. Use the `.env.example` file as a template and create your own `.env` file
3. Ensure `.env` is listed in `.gitignore` (it is by default)
4. Rotate your OAuth credentials periodically
5. Use separate OAuth applications for development and production

### Email Security

- Aerion stores emails locally on your device
- Use strong passwords for your email accounts
- Enable 2FA/MFA on your email accounts where possible
- For Gmail/Google Workspace: Use App-Specific Passwords or OAuth

### Data Storage

- Email data is stored in SQLite databases in your local data directory
- Ensure your device has appropriate security measures (disk encryption, screen lock, etc.)
- Back up your data regularly

## Security Features

Aerion includes the following security measures:

- **HTML Sanitization**: All HTML email content is sanitized before display to prevent XSS attacks
- **OAuth 2.0**: Secure authentication for Gmail and other OAuth-supporting providers
- **Local Storage**: Emails are stored locally, not on third-party servers
- **TLS/SSL**: All IMAP/SMTP connections use TLS encryption

## Known Limitations

- Aerion is currently in early stage active development
- Use at your own risk for sensitive communications
