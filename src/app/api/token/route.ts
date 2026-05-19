import { SignJWT } from "jose"
import { NextResponse } from "next/server"

import { auth } from "@/auth"

/**
 * GET /api/token
 *
 * Mint HS256 JWT untuk Go BE menggunakan `AUTH_SECRET` (shared secret).
 * FE memanggil endpoint ini sekali per session (lihat src/api/apiApp.ts) dan
 * attach token-nya sebagai `Authorization: Bearer ...` di setiap request ke BE.
 *
 * Lifetime token: 30 menit. FE auto-refresh kalau 401.
 */
export async function GET() {
  const session = await auth()
  if (!session?.user?.id || !session.user.email) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 })
  }

  const secret = process.env.AUTH_SECRET
  if (!secret || secret.length < 32) {
    console.error("[api/token] AUTH_SECRET missing or too short")
    return NextResponse.json({ error: "server misconfigured" }, { status: 500 })
  }

  const expiresInSeconds = 30 * 60
  const expiresAt = Math.floor(Date.now() / 1000) + expiresInSeconds

  const token = await new SignJWT({
    sub: session.user.id, // Google's "sub" — stable user identifier.
    email: session.user.email,
    name: session.user.name ?? "",
    picture: session.user.image ?? "",
  })
    .setProtectedHeader({ alg: "HS256", typ: "JWT" })
    .setIssuedAt()
    .setExpirationTime(expiresAt)
    .sign(new TextEncoder().encode(secret))

  return NextResponse.json({ token, expiresAt })
}
