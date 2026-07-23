# Security Policy

## Supported Versions

LRSolid is pre-1.0 research software. Security fixes are expected to target the current development branch unless a maintained release branch is explicitly announced.

| Version | Supported |
|---------|-----------|
| 0.1.x | Yes |
| < 0.1 | No |

## Reporting a Vulnerability

Please do not report security vulnerabilities through public issues.

Preferred reporting channels:

- Use GitHub private vulnerability reporting for this repository, if available.
- Otherwise, contact the maintainer privately through an established project or institutional channel.

Include:

- Affected version or commit.
- Steps to reproduce.
- Expected and actual impact.
- Any known workarounds.

Please avoid including real secrets, personal data, production credentials, or private learner records in reports.

## Scope

Security-sensitive areas include:

- Authentication and session handling.
- xAPI client credentials and scope enforcement.
- Multi-tenant access control between organisations, projects, and LRS instances.
- Statement, state, and profile data isolation.
- Export destinations and stored connection secrets.
