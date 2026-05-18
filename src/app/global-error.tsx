"use client"

interface GlobalErrorProps {
  error: Error & { digest?: string }
  reset: () => void
}

export default function GlobalError({ error, reset }: GlobalErrorProps) {
  console.error("[Global Error]", error)
  return (
    <html lang="id">
      <body>
        <div className="flex h-dvh flex-col items-center justify-center gap-4 p-4 text-center">
          <h1 className="text-2xl font-bold">Terjadi kesalahan</h1>
          <p className="max-w-md text-sm text-muted-foreground">
            Sesuatu terjadi yang tidak kami antisipasi. Coba lagi, atau kembali ke halaman utama.
          </p>
          <button
            onClick={reset}
            className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:opacity-90"
          >
            Coba lagi
          </button>
        </div>
      </body>
    </html>
  )
}
