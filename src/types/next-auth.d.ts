/* eslint-disable @typescript-eslint/no-unused-vars */
import type { DefaultSession } from "next-auth"

/**
 * Extend session.user dengan `id` (Google sub) supaya bisa diakses dari client.
 */
declare module "next-auth" {
  interface Session {
    user: {
      id: string
    } & DefaultSession["user"]
  }
}
