/**
 * Token cache util — fetch HS256 JWT dari /api/token (signed pakai AUTH_SECRET)
 * untuk dipakai sebagai Bearer token di request ke Go BE.
 *
 * Cache di module scope (single-tab), refresh 60 detik sebelum expiry.
 * Concurrent calls share the same in-flight promise.
 */

interface TokenResponse {
  token: string
  expiresAt: number // unix seconds
}

interface CachedToken {
  token: string
  expiresAt: number // unix seconds
}

let cached: CachedToken | null = null
let inFlight: Promise<CachedToken> | null = null

const SAFETY_WINDOW_SECONDS = 60

function isFresh(t: CachedToken | null): t is CachedToken {
  if (!t) return false
  const nowSec = Math.floor(Date.now() / 1000)
  return t.expiresAt - SAFETY_WINDOW_SECONDS > nowSec
}

async function fetchToken(): Promise<CachedToken> {
  const res = await fetch("/api/token", {
    method: "GET",
    credentials: "include",
    cache: "no-store",
  })
  if (!res.ok) {
    throw new Error(`Failed to fetch token: ${res.status}`)
  }
  const data = (await res.json()) as TokenResponse
  return { token: data.token, expiresAt: data.expiresAt }
}

/**
 * Returns a valid auth token. Caches in-memory; auto-refreshes when near expiry.
 * Safe to call concurrently — shares in-flight promise.
 */
export async function getAuthToken(): Promise<string> {
  if (isFresh(cached)) return cached.token
  if (inFlight)
    return inFlight.then((t) => t.token).catch(() => {
      inFlight = null
      return getAuthToken()
    })

  inFlight = fetchToken()
    .then((t) => {
      cached = t
      return t
    })
    .finally(() => {
      inFlight = null
    })
  const fresh = await inFlight
  return fresh.token
}

/** Force-clear cached token (use on 401 to trigger fresh fetch on next call). */
export function clearAuthToken() {
  cached = null
}
