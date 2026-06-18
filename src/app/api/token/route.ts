import { SignJWT } from "jose"
import { getToken } from "next-auth/jwt"
import { NextResponse, type NextRequest } from "next/server"

import { auth } from "@/auth"

/**
 * GET /api/token
 *
 * Mint HS256 JWT untuk Go BE menggunakan `AUTH_SECRET` (shared secret).
 * FE memanggil endpoint ini sekali per session (lihat src/api/apiApp.ts) dan
 * attach token-nya sebagai `Authorization: Bearer ...` di setiap request ke BE.
 *
 * Minggu 9: Google access token di-forward via `google_access_token` claim.
 * BE pakai untuk call Calendar / Gmail APIs atas nama user. Token di-refresh
 * di FE auth.ts jwt callback (Auth.js v5) — di sini cuma forward yang ada.
 *
 * Lifetime token: 30 menit. FE auto-refresh kalau 401.
 */
export async function GET(req: NextRequest) {
  const session = await auth()
  if (!session?.user?.id || !session.user.email) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 })
  }

  const secret = process.env.AUTH_SECRET
  if (!secret || secret.length < 32) {
    console.error("[api/token] AUTH_SECRET missing or too short")
    return NextResponse.json({ error: "server misconfigured" }, { status: 500 })
  }

  // `session()` callback nggak expose access_token (privacy default Auth.js).
  // Kita perlu raw JWT via `getToken` untuk access googleAccessToken claim.
  const raw = await getToken({ req, secret })
  const googleAccessToken = (raw?.googleAccessToken as string | undefined) ?? ""

  const expiresInSeconds = 30 * 60
  const expiresAt = Math.floor(Date.now() / 1000) + expiresInSeconds

  const token = await new SignJWT({
    sub: session.user.id,
    email: session.user.email,
    name: session.user.name ?? "",
    picture: session.user.image ?? "",
    google_access_token: googleAccessToken,
  })
    .setProtectedHeader({ alg: "HS256", typ: "JWT" })
    .setIssuedAt()
    .setExpirationTime(expiresAt)
    .sign(new TextEncoder().encode(secret))

  return NextResponse.json({ token, expiresAt })
}
