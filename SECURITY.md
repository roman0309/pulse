# Security Policy

## Reporting a vulnerability

If you discover a security vulnerability in Pulse, **please do not open a public
issue.** Report it privately so we can fix it before it is widely known.

- Preferred: open a [GitHub Security Advisory](https://github.com/roman0309/pulse/security/advisories/new) (private).
- Or email **risbcian@gmail.com** with details.

Please include:

- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- affected version / commit, and
- any suggested remediation.

We aim to acknowledge reports within **72 hours** and to provide a fix or
mitigation timeline after triage. We will credit reporters who wish to be named
once a fix is released.

## Supported versions

Pulse is pre-1.0; security fixes land on the latest `main` and the most recent
release tag.

## Hardening reminders for operators

Pulse ships with development defaults. Before exposing an instance:

- Rotate `JWT_SECRET` and `JWT_REFRESH_SECRET` (use long random values).
- Change all database passwords; do not run datastores with default credentials.
- Terminate TLS at your reverse proxy and restrict `CORS_ORIGINS`.
- Treat ingest keys as secrets; rotate/revoke them from the Connect page.

See [DEPLOYMENT.md](DEPLOYMENT.md) for the full production checklist.
