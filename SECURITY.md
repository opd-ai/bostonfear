# Security Policy

## Reporting a Security Vulnerability

If you discover a security vulnerability in BostonFear, please report it responsibly to the maintainers rather than disclosing it publicly.

### How to Report

Please email security concerns to the project maintainers via the GitHub repository's security contact or open a **private security advisory** on GitHub:

1. Go to the [GitHub Security Advisory page](https://github.com/opd-ai/bostonfear/security/advisories)
2. Click "Report a vulnerability"
3. Provide detailed information about the vulnerability and steps to reproduce (if applicable)

**Please include**:
- A clear description of the vulnerability
- Steps to reproduce (if applicable)
- Estimated impact (e.g., does it affect authentication, data integrity, availability?)
- Any suggested fixes or mitigations

### Response Timeline

- **Initial acknowledgment**: Within 2 business days
- **Status updates**: At least weekly while investigating
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days

## Security Considerations for Users

### WebSocket Origin Validation

By default, BostonFear accepts WebSocket upgrades from any origin (permissive mode for local development). **For production deployments**, you should restrict allowed origins:

```go
gameEngine.SetAllowedOrigins([]string{
    "yourdomain.com",
    "www.yourdomain.com",
})
```

If `AllowedOrigins` is not set or is empty, the server accepts all origins.

### Connection Inactivity Timeout

The server applies a 30-second inactivity timeout on all connections. If a player sends no messages for 30 seconds, the doom counter is incremented and the connection is closed. This prevents blocked connections from affecting game state indefinitely.

### Recommended Practices

1. **HTTPS in Production**: Deploy the WebSocket server behind a reverse proxy (nginx, HAProxy) with TLS/SSL enabled
2. **Origin Restrictions**: Always configure `SetAllowedOrigins()` for production domains
3. **Rate Limiting**: Consider implementing rate limiting on the HTTP endpoint to prevent connection floods
4. **Monitoring**: Use the `/metrics` and `/health` endpoints to monitor connection patterns and detect anomalies
5. **Dependencies**: Keep Go and dependencies updated regularly (use `go get -u` with caution and test thoroughly)

## No Warranties

This project is provided "as-is" without warranties. While we strive for reliability, BostonFear is an educational framework demonstrating game mechanics and WebSocket architecture, not a hardened production platform.

## Licensing

BostonFear is licensed under the MIT License. See [LICENSE](LICENSE) for details.
