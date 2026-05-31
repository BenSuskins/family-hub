# Security Policy

## Reporting a Vulnerability

If you believe you have found a security vulnerability in Family Hub, please
report it privately. **Do not open a public issue or pull request** for
security problems, as that discloses the vulnerability before a fix is
available.

To report a vulnerability:

1. Go to the [**Security** tab](../../security) of this repository.
2. Click **Report a vulnerability** to open a private security advisory.
3. Provide as much detail as you can, including:
   - A description of the vulnerability and its potential impact.
   - Steps to reproduce (proof-of-concept, affected endpoints/components).
   - The version, commit, or deployment where you observed the issue.
   - Any suggested remediation, if you have one.

Reports are handled privately through GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
so a fix can be prepared before public disclosure.

## What to Expect

- **Acknowledgement** of your report as soon as we are able to review it.
- An assessment of the report and, if confirmed, a plan for a fix.
- Coordinated disclosure: we will work with you on timing and will credit
  you in the advisory once a fix is released, unless you prefer to remain
  anonymous.

## Scope

This policy covers the code in this repository, including:

- The Go server (`server/`)
- The iOS app (`ios/`)
- The Home Assistant integration (`home-assistant/`)

Please note that Family Hub is a self-hosted application. Misconfiguration of
your own deployment (for example, exposing the service without authentication,
or a misconfigured reverse proxy or OIDC provider) is the responsibility of the
operator and is generally outside the scope of this policy.

## Supported Versions

Family Hub is developed on a rolling basis. Security fixes are applied to the
latest release and the `main` branch. Please ensure you are running the most
recent version before reporting an issue.
