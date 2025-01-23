# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

## Reporting a Vulnerability

1. **Do NOT** open a public issue
2. Email security@wolfai.com with:
   - Description of vulnerability
   - Steps to reproduce
   - Potential impact
3. Expect response within 48 hours
4. Security team will:
   - Confirm receipt
   - Assess severity
   - Develop fix
   - Issue advisory

## Security Measures

- Go 1.21+ required for security patches
- Dependencies auto-updated via Dependabot
- Regular security audits
- Encrypted data in transit and at rest
- Rate limiting on all endpoints
- Input validation and sanitization
- Prepared statements for SQL
- Security headers:
  - CSP
  - HSTS
  - XSS Protection
  - Frame Options

## Development Guidelines

- Use latest stable dependencies
- Run `make security-check` before commits
- Enable 2FA for repository access
- Review security advisories weekly
