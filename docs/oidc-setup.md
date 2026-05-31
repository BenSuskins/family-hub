# OIDC Client Setup

Family Hub uses a **single public OIDC client with PKCE** for both the web UI
and the iOS app — no client secret, two redirect URIs. The iOS app discovers
the issuer and client ID at runtime via `GET /api/client-config`, so you only
configure those values on the server.

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
          - https://hub.example.com/auth/callback    # web
          - familyhub://callback                      # iOS
        scopes: [openid, profile, email]
        grant_types: [authorization_code]
        response_types: [code]
        userinfo_signed_response_alg: none
```

## Keycloak

1. Create a new client with **Client ID** `family-hub`.
2. Set **Client authentication** to `OFF` (public client).
3. Enable **Standard flow** only.
4. Under **Valid redirect URIs** add `https://hub.example.com/auth/callback` and `familyhub://callback` (for iOS).
5. Under **Advanced → Proof Key for Code Exchange Code Challenge Method** select `S256`.
6. Set `OIDC_ISSUER` to `https://<keycloak-host>/realms/<your-realm>`.

## Auth0

1. Create a **Single Page Application** (SPA) — this gives you a public PKCE client.
2. Under **Allowed Callback URLs** add `https://hub.example.com/auth/callback`.
3. Under **Allowed Web Origins** add `https://hub.example.com`.
4. Set `OIDC_ISSUER` to `https://<your-tenant>.auth0.com/`.
5. Set `OIDC_CLIENT_ID` to the Auth0 Client ID shown in the application settings.

> Auth0's free tier limits to 7,500 monthly active users — more than enough for a family.
