# Troubleshooting

> Common issues and fixes for Family Hub.

## Setup

**Server exits because `OIDC_ISSUER` is missing**

Cause: the API needs the issuer to publish mobile client configuration and
validate token exchange requests.

Fix: set `OIDC_ISSUER` and `OIDC_CLIENT_ID` in the server environment.

## Auth

**iOS app cannot exchange its OIDC token**

Cause: the app cannot reach `POST /api/auth/exchange`, or the provider userinfo
endpoint rejects the OIDC bearer token.

Fix: check the server URL in the app, the reverse proxy route, and the provider
userinfo endpoint. See [OIDC Client Setup](oidc-setup.md).

**API request returns `401`**

Cause: the request has no valid Family Hub API token. An OIDC access token is
only accepted by the one-time token exchange route.

Fix: exchange a fresh OIDC access token at `/api/auth/exchange`, then send the
returned token as `Authorization: Bearer <token>`.

**First user is not admin**

Cause: only the first user created through token exchange is auto-promoted.

Fix: promote the account through an existing admin API token or update the
user role in SQLite.

## iOS

**iOS app shows "Unable to connect"**

Cause: the device cannot reach `GET /api/client-config` or the API server.

Fix: check the server URL in the app, DNS, TLS, firewall rules, and reverse
proxy routing.

## Wall display

**The display keeps showing its pairing code**

Cause: an admin has not approved the pairing request, the code expired, or the
display cannot reach the API.

Fix: open Wall Displays in the iOS app, scan the current code, approve the
display, and check the API and browser network route.

**A section does not appear on the wall display**

Cause: that section is disabled for the paired display, or it has no current
household data to show.

Fix: open the display in the iOS app and enable the section. Check the relevant
household data if the section remains empty.
