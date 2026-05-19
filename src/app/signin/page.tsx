import { redirect } from "next/navigation"

import { auth, signIn } from "@/auth"

interface SigninPageProps {
  searchParams: Promise<{ callbackUrl?: string }>
}

export default async function SigninPage(props: SigninPageProps) {
  const session = await auth()
  if (session?.user) {
    redirect("/chat")
  }

  const { callbackUrl } = await props.searchParams

  return (
    <main className="flex min-h-dvh items-center justify-center bg-background px-6">
      <div className="w-full max-w-sm space-y-6 rounded-2xl border border-border bg-card p-8 shadow-sm">
        <div className="space-y-2 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">Personal Chat AI by Aulia</h1>
          <p className="text-sm text-muted-foreground">
            Sign in untuk akses chat history dan settings kamu.
          </p>
        </div>

        <form
          action={async () => {
            "use server"
            await signIn("google", { redirectTo: callbackUrl ?? "/chat" })
          }}
        >
          <button
            type="submit"
            className="flex w-full items-center justify-center gap-3 rounded-lg border border-border bg-background px-4 py-2.5 text-sm font-medium shadow-sm transition-colors hover:bg-accent"
          >
            <GoogleIcon className="h-4 w-4" />
            Sign in with Google
          </button>
        </form>

        <p className="text-center text-xs text-muted-foreground">
          By signing in kamu setuju kalau chat history disimpan di Neon Postgres.
        </p>
      </div>
    </main>
  )
}

function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09Z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.99.66-2.25 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84A10.99 10.99 0 0 0 12 23Z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.09a6.6 6.6 0 0 1 0-4.18V7.07H2.18a11 11 0 0 0 0 9.86l3.66-2.84Z"
      />
      <path
        fill="#EA4335"
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84C6.71 7.31 9.14 5.38 12 5.38Z"
      />
    </svg>
  )
}
