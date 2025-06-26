# Security Policy

## Supported Versions

We actively support the following versions of GHA Pinner with security updates:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

## Reporting a Vulnerability

We take the security of GHA Pinner seriously. If you discover a security vulnerability, please follow these steps:

### 🔒 Private Disclosure

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report security vulnerabilities by:

1. **Email**: Send details to the maintainers (check the repository for current contact information)
2. **GitHub Security Advisory**: Use GitHub's private vulnerability reporting feature
3. **Direct Message**: Contact maintainers directly through secure channels

### 📝 What to Include

When reporting a vulnerability, please include:

- **Description**: A clear description of the vulnerability
- **Impact**: What kind of vulnerability it is and what an attacker could do
- **Steps to Reproduce**: Detailed steps to reproduce the issue
- **Affected Versions**: Which versions of GHA Pinner are affected
- **Proof of Concept**: If possible, include a minimal proof of concept
- **Suggested Fix**: If you have ideas for how to fix the issue

### 🕐 Response Timeline

We aim to respond to security vulnerability reports within:

- **Initial Response**: 48 hours
- **Status Update**: 7 days
- **Resolution Timeline**: We'll provide an estimated timeline based on severity

### 🛡️ Security Best Practices

When using GHA Pinner:

1. **Keep Updated**: Always use the latest version
2. **Verify Integrity**: Verify checksums when downloading releases
3. **Review Changes**: Review the changelog for security-related updates
4. **Environment Security**: Ensure your development environment is secure
5. **Token Security**: Properly secure your GitHub tokens and credentials

### 🏆 Recognition

We appreciate security researchers who help keep GHA Pinner secure:

- Researchers who responsibly disclose vulnerabilities will be credited in release notes (unless they prefer to remain anonymous)
- We may feature significant contributions in our security acknowledgments

### 📚 Additional Resources

- [GitHub Security Best Practices](https://docs.github.com/en/code-security)
- [Go Security Best Practices](https://golang.org/doc/security/)
- [OWASP Security Guidelines](https://owasp.org/)

## Security Features

GHA Pinner includes several security features:

- **Dependency Pinning**: Pins GitHub Actions to specific commit hashes
- **Hash Verification**: Verifies action integrity through commit hashes
- **Minimal Permissions**: Requires only necessary GitHub permissions
- **Audit Trail**: Provides clear logs of all pinning operations

## Contact

For security-related questions or concerns, please reach out through the channels mentioned above.

Thank you for helping keep GHA Pinner secure! 🔒
