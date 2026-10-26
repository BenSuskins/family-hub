# OIDC Client Setup

Family Hub uses a **public OIDC client with PKCE** for iOS sign-in. The Go API
does not run the OIDC browser flow and stores no client secret. The iOS app
discovers the issuer and client ID via `GET /api/client-config`.

Pick your provider below.

## Authelia

```yaml
identity_providers:
  oidc:
    clients:
      - client_id: family-hub
        client_name: Family Hub
        public: true
        token_endpoint_auth_method: none
        require_pkce: true
        pkce_challenge_method: S256
        redirect_uris:
          - familyhub://callback
        scopes: [openid, profile, email]
        grant_types: [authorization_code]
        response_types: [code]
        userinfo_signed_response_alg: none
```

## Keycloak

1. Create a new client with **Client ID** `family-hub`.
2. Set **Client authentication** to `OFF` (public client).
3. Enable **Standard flow** only.
4. Under **Valid redirect URIs** add `familyhub://callback`.
5. Under **Advanced → Proof Key for Code Exchange Code Challenge Method** select `S256`.
6. Set `OIDC_ISSUER` to `https://<keycloak-host>/realms/<your-realm>`.

## Auth0

1. Create a **Single Page Application** (SPA) — this gives you a public PKCE client.
2. Under **Allowed Callback URLs** add `familyhub://callback`.
3. Add the iOS app as an allowed origin if your provider requires it.
4. Set `OIDC_ISSUER` to `https://<your-tenant>.auth0.com/`.
5. Set `OIDC_CLIENT_ID` to the Auth0 Client ID shown in the application settings.

> Auth0's free tier limits to 7,500 monthly active users — more than enough for a family.

## Web client

The web app uses a separate public authorization-code client with PKCE S256.
Add `https://<web-host>/auth/callback` as an allowed redirect URI and the web
origin as an allowed origin. Use the client ID returned by
`GET /api/client-config`; do not put a client secret in the web build.

Set `NEXT_PUBLIC_FAMILY_HUB_API_URL` to the Go API origin when building the
web app. Set the API server's `WEB_ORIGIN` to the exact web origin, including
scheme and port. The API permits browser requests only from this origin.

The browser exchanges its OIDC bearer at
`POST /api/auth/exchange?client=web`. This creates or rotates the Web App token
without revoking the user's iOS token.
