# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest (`main`) | ✅ |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Use [GitHub private vulnerability reporting](https://github.com/kengou/simplewebhook/security/advisories/new) to report a vulnerability confidentially. You will receive a response within 5 business days.

Please include:

- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept
- The version or commit affected

## Security Considerations

- Webhook payloads are authenticated via HMAC-SHA256 (`X-Hub-Signature-256`) when `WEBHOOK_SECRET` is set
- Request bodies are capped at 1 MiB to prevent memory exhaustion
- The container runs as a non-root user (`65532`) with a read-only filesystem and all capabilities dropped
- No external runtime dependencies; stdlib only
