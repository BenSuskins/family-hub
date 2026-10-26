import { readFile } from "node:fs/promises";
import { join } from "node:path";

export async function GET() {
  if (process.env.NODE_ENV !== "development") return Response.json({ error: "Local display mocks are only available in development." }, { status: 404 });
  const script = await readFile(join(process.cwd(), "node_modules/msw/lib/mockServiceWorker.js"), "utf8");
  return new Response(script, {
    headers: {
      "Cache-Control": "no-store",
      "Content-Type": "application/javascript; charset=utf-8",
      "Service-Worker-Allowed": "/",
    },
  });
}
