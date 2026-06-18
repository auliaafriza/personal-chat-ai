import NextAuth from "next-auth"
import Google from "next-auth/providers/google"

/**
 * Auth.js v5 (NextAuth) — Google OAuth dengan extended scopes (Minggu 9).
 *
 * Scopes:
 *   - openid, email, profile (default)
 *   - calendar (read+write events di primary Calendar)
 *   - gmail.readonly (search + read inbox)
 *
 * Access token disimpan di session JWT supaya BE bisa proxy ke Google APIs.
 * Auto-refresh via refresh_token saat expired (1 jam). Required:
 *   access_type=offline + prompt=consent → guaranteed refresh_token issued.
 *
 * NOTE: AUTH_SECRET di-share dengan backend Go (lihat backend/.env). HS256.
 */

const GOOGLE_SCOPES = [
  "openid",
  "email",
  "profile",
  "https://www.googleapis.com/auth/calendar",
  "https://www.googleapis.com/auth/gmail.readonly",
].join(" ")

// 60s safety window — refresh kalau expiring dalam 1 menit
const REFRESH_BUFFER_SECONDS = 60

interface GoogleRefreshResponse {
  access_token: string
  expires_in: number
  refresh_token?: string
  error?: string
}

async function refreshGoogleAccessToken(refreshToken: string): Promise<{
  accessToken: string
  expiresAt: number
  refreshToken?: string
} | null> {
  try {
    const res = await fetch("https://oauth2.googleapis.com/token", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        client_id: process.env.AUTH_GOOGLE_ID ?? "",
        client_secret: process.env.AUTH_GOOGLE_SECRET ?? "",
        refresh_token: refreshToken,
        grant_type: "refresh_token",
      }),
    })
    const data = (await res.json()) as GoogleRefreshResponse
    if (!res.ok || data.error || !data.access_token) {
      console.error("[auth] Google refresh failed:", data)
      return null
    }
    return {
      accessToken: data.access_token,
      expiresAt: Math.floor(Date.now() / 1000) + data.expires_in,
      refreshToken: data.refresh_token, // Google sometimes rotates
    }
  } catch (err) {
    console.error("[auth] Google refresh exception:", err)
    return null
  }
}

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [
    Google({
      clientId: process.env.AUTH_GOOGLE_ID,
      clientSecret: process.env.AUTH_GOOGLE_SECRET,
      authorization: {
        params: {
          scope: GOOGLE_SCOPES,
          access_type: "offline",
          prompt: "consent",
        },
      },
    }),
  ],
  pages: {
    signIn: "/signin",
  },
  callbacks: {
    async jwt({ token, profile, account }) {
      // Initial sign-in: salin profile + tokens.
      if (account) {
        token.googleAccessToken = account.access_token
        token.googleRefreshToken = account.refresh_token
        token.googleExpiresAt = account.expires_at as number | undefined
      }
      if (profile) {
        token.sub = profile.sub ?? token.sub
        token.email = profile.email ?? token.email
        token.name = (profile.name as string | undefined) ?? token.name
        token.picture = (profile.picture as string | undefined) ?? token.picture
      }

      // Refresh kalau access token expired/expiring.
      const expiresAt = token.googleExpiresAt as number | undefined
      const refreshToken = token.googleRefreshToken as string | undefined
      const now = Math.floor(Date.now() / 1000)
      if (expiresAt && refreshToken && expiresAt - REFRESH_BUFFER_SECONDS < now) {
        const refreshed = await refreshGoogleAccessToken(refreshToken)
        if (refreshed) {
          token.googleAccessToken = refreshed.accessToken
          token.googleExpiresAt = refreshed.expiresAt
          if (refreshed.refreshToken) {
            token.googleRefreshToken = refreshed.refreshToken
          }
        } else {
          // Refresh failed — clear access supaya BE tahu user harus re-auth.
          token.googleAccessToken = undefined
        }
      }

      return token
    },
    async session({ session, token }) {
      if (session.user) {
        session.user.id = (token.sub as string) ?? ""
        session.user.email = (token.email as string) ?? session.user.email
        session.user.name = (token.name as string) ?? session.user.name
        session.user.image = (token.picture as string) ?? session.user.image
      }
      return session
    },
  },
  session: { strategy: "jwt" },
  trustHost: true,
})
