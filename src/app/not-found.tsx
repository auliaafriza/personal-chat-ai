import Link from "next/link"

export default function NotFound() {
  return (
    <div className="flex h-dvh flex-col items-center justify-center gap-4">
      <h1 className="text-3xl font-bold">404</h1>
      <p className="text-muted-foreground">Halaman tidak ditemukan</p>
      <Link href="/chat" className="text-primary underline underline-offset-2">
        Kembali ke chat
      </Link>
    </div>
  )
}
