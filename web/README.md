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

Open `http://localhost:3000`. Set `NEXT_PUBLIC_FAMILY_HUB_API_URL` to the Family Hub Go API origin. Configure `WEB_ORIGIN` on the Go server to the exact web origin so CORS allows the display to call the API.

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_FAMILY_HUB_API_URL` | yes | — | Base URL of the Go API, for example `https://family.example.com`. |

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
