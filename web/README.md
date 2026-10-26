# Family Hub Web

> A shared, simple view of the family day, with a wall display for the home.

## Overview

Family Hub Web is the browser client for a single Family Hub server installation. It uses the existing Go API for sign-in and shared household data. A paired wall display gets a limited display token and can show the day without exposing member account credentials.

## Features

- Today, calendar, chores, meals, recipes, shopping, and inventory pages.
- Admin controls for household settings and paired wall displays.
- Full-screen wall display with a launcher for shared household tasks.
- Responsive layout for phones, tablets, and larger screens.

## Tech Stack

- TypeScript 7 and React 19.
- Next.js 16 App Router.
- Bun 1.4.4 for package management and scripts.
- CSS and `qrcode.react` for the interface and pairing QR code.

## Project Structure

```
web/
├── src/app/       # Routes and shared page components
├── public/        # Static assets and design prototypes
└── package.json   # Bun scripts and dependencies
```

## Getting Started

### Prerequisites

- Bun 1.4.4 or a compatible Bun 1.x release.
- A running Family Hub Go server.

### Setup

```bash
cd web
bun install --frozen-lockfile
bun run dev
```

Open `http://localhost:3000`. The app should show the sign-in page or the current-day view after sign-in.

Configure the Go server's `WEB_ORIGIN` to the exact browser origin, such as `http://localhost:3000`. The web client calls the Go API directly, so the API must allow that origin through CORS.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_FAMILY_HUB_API_URL` | yes | — | Base URL of the Go API, for example `https://family.example.com`. |

Set this value when building and running the web app. Configure the OIDC provider's callback URL as `<web-origin>/auth/callback`. See [OIDC setup](../docs/oidc-setup.md).

## Commands

| Command | Purpose |
|---------|---------|
| `bun run dev` | Run the local development server. |
| `bun run lint` | Check source with Oxlint. |
| `bun run build` | Build the production app. |
| `bun run start` | Serve the production build. |

## Documentation

- [Web product requirements](../docs/web-frontend-product-requirements.md)
- [Web design](../docs/web-design.md)
- [API endpoints](../docs/endpoints.md)
- [OIDC setup](../docs/oidc-setup.md)

## Deployment

Build and deploy this app as a separate web service from the Go API. Provide `NEXT_PUBLIC_FAMILY_HUB_API_URL` at build time and set `WEB_ORIGIN` on the Go server to the public web origin. This client does not create a separate household or database.

## Security

Web sign-in exchanges the OIDC identity token for a web-scoped Family Hub API token. The API stores that token in browser local storage. Display tokens have separate read and limited write routes; they cannot call member or admin routes. Pairing requires an authenticated admin to approve the display. See [API endpoints](../docs/endpoints.md) and [OIDC setup](../docs/oidc-setup.md).

## License

See the repository [LICENSE](../LICENSE).
