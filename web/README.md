# Family Hub Web

The web service serves the QR-paired household wall display. Before pairing, it shows a QR code for approval in the iOS app. After approval, it shows the shared dashboard with the sections enabled for that display.

## Tech stack

- TypeScript 7 and React 19
- Next.js 16 App Router
- Bun 1.4.4
- CSS and `qrcode.react` for pairing

## Run locally

```bash
cd web
bun install --frozen-lockfile
bun run dev
```

Open `http://localhost:3000`. Without `NEXT_PUBLIC_FAMILY_HUB_API_URL`, local development uses the JSON-backed display mock. To use a real API, set `NEXT_PUBLIC_FAMILY_HUB_API_URL` to its origin and configure `WEB_ORIGIN` on the Go server to allow the web origin.

### Local display mock

When the API URL is unset, the display starts an MSW service worker in the browser. MSW serves the fixture in `mocks/display.json` and keeps write changes in memory. The mock includes pairing, display data, and display write actions. Changes reset when the browser reloads. Edit the fixture to change the sample household.

Use `http://localhost:3000/display?mock=paired` to skip the approval step in E2E tests. The regular `/display` route shows the pairing screen and waits for local approval.

To approve a pairing in a browser, open the developer console after the pairing code appears and run:

```js
await fetch("/api/local-mocks/pairing/approve", { method: "POST" })
```

E2E tests can call the same endpoint through the page context after waiting for the pairing code:

```ts
await page.evaluate(() => fetch("/api/local-mocks/pairing/approve", { method: "POST" }));
```

Reset mock data and pairing approval from the browser console with:

```js
await fetch("/api/local-mocks/reset", { method: "POST" })
```

Clear the display's local storage to return to the pairing screen. MSW is enabled only in local development with no API URL configured. Do not use the mock token or fixtures in a deployed environment.

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_FAMILY_HUB_API_URL` | production | — | Base URL of the Go API, for example `https://family.example.com`. Leave unset to use the local development mock. |

## Commands

| Command | Purpose |
|---------|---------|
| `bun run dev` | Run the local Next.js server. |
| `bun run lint` | Check source with Oxlint. |
| `bun run build` | Build the production app. |
| `bun run start` | Serve the production build. |

## Deployment

Build and deploy this app as a separate service from the Go API. The Docker image uses standalone output and includes only files needed to run the display. The `Web image` workflow publishes `ghcr.io/bensuskins/family-hub-web:latest` and a commit SHA tag after changes merge to `main`.

## Security

The display uses its own revocable display-scoped token. It cannot call member or admin routes. iOS admins pair, configure, and disconnect displays. See [API endpoints](../docs/endpoints.md) and the [shared design guide](../docs/DESIGN.md).

## License

See the repository [LICENSE](../LICENSE).
