import { NextResponse } from "next/server"

import { auth } from "@/auth"

/**
 * Middleware Auth.js v5 — guard semua route kecuali:
 *   - `/` (landing page publik, Minggu 12)
 *   - `/signin`
 *   - `/api/auth/*`
 *
 * Kalau belum login, redirect protected route ke `/signin?callbackUrl=...`.
 * Kalau udah login dan hit `/` atau `/signin`, redirect ke `/chat`.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export default auth((req: any) => {
  const { nextUrl } = req
  const isLoggedIn = !!req.auth
  const isAuthRoute = nextUrl.pathname.startsWith("/api/auth")
  const isSigninPage = nextUrl.pathname === "/signin"
  const isLandingPage = nextUrl.pathname === "/"

  if (isAuthRoute) return NextResponse.next()

  // Landing page: publik. Kalau udah login, langsung ke /chat.
  if (isLandingPage) {
    if (isLoggedIn) {
      return NextResponse.redirect(new URL("/chat", nextUrl))
    }
    return NextResponse.next()
  }

  if (isSigninPage) {
    if (isLoggedIn) {
      return NextResponse.redirect(new URL("/chat", nextUrl))
    }
    return NextResponse.next()
  }

  if (!isLoggedIn) {
    const callbackUrl = encodeURIComponent(nextUrl.pathname + nextUrl.search)
    return NextResponse.redirect(new URL(`/signin?callbackUrl=${callbackUrl}`, nextUrl))
  }

  return NextResponse.next()
})

export const config = {
  // Skip Next.js internals and static files.
  matcher: ["/((?!_next/static|_next/image|favicon.ico|.*\\..*).*)"],
}
