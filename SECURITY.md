# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in this project, please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Email: **rico.goerlitz@gmail.com**
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact

We will respond within **48 hours** and work with you to resolve the issue before any public disclosure.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.x.x   | ✅ Latest only |

## Security Best Practices

This service implements several security measures:

- **Session encryption** — All cached session data is encrypted with AES-256
- **Secret management** — Credentials stored in Azure KeyVault / HashiCorp Vault
- **No credential logging** — Secrets are never logged
- **gosec scanning** — Automated security scanning via golangci-lint
- **CodeQL analysis** — GitHub-native security analysis on every push
- **Dependency scanning** — Dependabot monitors for vulnerable dependencies
