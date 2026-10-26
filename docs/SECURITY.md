# Security Policy

## Reporting a Vulnerability

Please use [GitHub's private vulnerability reporting](https://github.com/bensuskins/family-hub/security/advisories/new) to report security issues. Do not open a public issue.

We will acknowledge the report and provide an expected timeline for a fix. Once resolved, a security advisory will be published.

## Scope

This policy covers code in this repository: the Go server (`server/`), iOS app
(`ios/`), and web client (`web/`). Misconfiguration of a self-hosted
deployment (exposed service without auth, misconfigured reverse proxy, or
misconfigured OIDC provider) is the operator's responsibility.
