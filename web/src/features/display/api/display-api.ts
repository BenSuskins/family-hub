import { displayStoragePrefix } from "@/lib/display-logic";
import { getApiBaseUrl } from "@/lib/api";

export const apiBase = getApiBaseUrl();
export const localMocksEnabled = process.env.NODE_ENV === "development" && !process.env.NEXT_PUBLIC_FAMILY_HUB_API_URL;
const storagePrefix = displayStoragePrefix(apiBase);
export const displayTokenKey = `${storagePrefix}.token`;
export const snapshotKey = (id: string) => `${storagePrefix}.snapshot.${encodeURIComponent(id)}`;

export class DisplayAccessRevoked extends Error {}

export function readStorage(key: string) {
  try { return window.localStorage.getItem(key); } catch { return null; }
}

export function writeStorage(key: string, value: string) {
  try { window.localStorage.setItem(key, value); return true; } catch { return false; }
}

export function removeStorage(key: string) {
  try { window.localStorage.removeItem(key); } catch { /* Browser storage can be disabled. */ }
}

export async function displayRequest<T>(path: string, token: string): Promise<T> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 8_000);
  try {
    const response = await fetch(apiBase + path, { headers: { Authorization: `Bearer ${token}` }, signal: controller.signal });
    if (response.status === 401 || response.status === 403) throw new DisplayAccessRevoked();
    if (!response.ok) throw new Error(`${path} returned ${response.status}`);
    return await response.json() as T;
  } finally { window.clearTimeout(timeout); }
}
