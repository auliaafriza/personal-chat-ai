import NextAuth from "next-auth"
import Google from "next-auth/providers/google"

/**
 * Auth.js v5 (NextAuth) — Google OAuth.
 *
 * Token shape kita simpan (`jwt` callback) supaya /api/token bisa mint
 * HS256 JWT untuk Go BE pakai field yang sama (sub, email, name, picture).
 *
 * NOTE: AUTH_SECRET di-share dengan backend Go (lihat backend/.env). HS256.
 */
export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [
    Google({
      clientId: process.env.AUTH_GOOGLE_ID,
      clientSecret: process.env.AUTH_GOOGLE_SECRET,
    }),
  ],
  pages: {
    signIn: "/signin",
  },
  callbacks: {
    async jwt({ token, profile }) {
      // Pas first login: salin field profile Google ke token.
      // Field-field ini yang nanti kita re-sign sebagai HS256 untuk Go BE.
      if (profile) {
        token.sub = profile.sub ?? token.sub
        token.email = profile.email ?? token.email
        token.name = (profile.name as string | undefined) ?? token.name
        token.picture = (profile.picture as string | undefined) ?? token.picture
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
