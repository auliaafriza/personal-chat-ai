import type { Metadata } from "next"
import type { ReactNode } from "react"

import { Inter } from "next/font/google"
import { Toaster } from "sonner"

import { QueryProvider } from "@/providers/QueryProvider"
import { SessionProvider } from "@/providers/SessionProvider"
import { ThemeProvider } from "@/providers/ThemeProvider"

import "./globals.css"

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" })

export const metadata: Metadata = {
  title: "Personal Chat AI by Aulia — Your AI Assistant",
  description: "Chat assistant pribadi untuk dokumen, kode, dan produktivitas.",
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="id" suppressHydrationWarning>
      <body className={`${inter.variable} font-sans antialiased`}>
        <SessionProvider>
          <ThemeProvider>
            <QueryProvider>
              {children}
              <Toaster position="top-center" richColors />
            </QueryProvider>
          </ThemeProvider>
        </SessionProvider>
      </body>
    </html>
  )
}
