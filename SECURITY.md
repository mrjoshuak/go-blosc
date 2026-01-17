# Security Policy

## Supported Versions

| Version | Supported              |
| ------- | ---------------------- |
| 1.x     | ✅ Active support      |
| < 1.0   | ❌ No longer supported |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in go-blosc, please report it responsibly.

### How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please use GitHub's private vulnerability reporting feature:

1. Go to the [Security tab](../../security) of this repository
2. Click "Report a vulnerability"
3. Fill out the form with details about the vulnerability

### What to Include

| Information   | Description                                |
| ------------- | ------------------------------------------ |
| Description   | What the vulnerability is and how it works |
| Reproduction  | Steps to reproduce the issue               |
| Impact        | Potential security impact if exploited     |
| Suggested fix | Optional: any ideas for remediation        |

### Response Timeline

| Milestone          | Timeframe                 |
| ------------------ | ------------------------- |
| Acknowledgment     | Within 48 hours           |
| Initial assessment | Within 7 days             |
| Resolution         | Coordinated with reporter |

### Disclosure Policy

- We will coordinate with you on disclosure timing
- We will credit reporters in security advisories (unless you prefer anonymity)
- We ask that you give us reasonable time to address the issue before public disclosure

## Security Best Practices

When using go-blosc:

- Always validate Blosc data from untrusted sources
- Be aware that malformed Blosc data could cause excessive memory allocation
- Use appropriate resource limits when processing data from untrusted sources
- The library includes bounds checking and size limits to prevent common attacks

## Scope

This security policy applies to the go-blosc library code. Issues in the Blosc format specification or the C Blosc library should be reported to the [Blosc project](https://github.com/Blosc/c-blosc).
