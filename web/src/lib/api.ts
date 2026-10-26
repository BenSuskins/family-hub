export function getApiBaseUrl(): string {
  return process.env.NEXT_PUBLIC_FAMILY_HUB_API_URL?.replace(/\/$/, "") ?? "";
}
