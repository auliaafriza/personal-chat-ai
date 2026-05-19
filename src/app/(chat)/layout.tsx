import type { ReactNode } from "react"

import { redirect } from "next/navigation"

import { auth } from "@/auth"
import { Sidebar } from "@/components/layout/Sidebar"

export default async function ChatLayout({ children }: { children: ReactNode }) {
  const session = await auth()
  if (!session?.user) {
    redirect("/signin")
  }

  return (
    <div className="flex h-dvh bg-background">
      {/* Desktop sidebar — hidden on mobile, mobile drawer di-render dari ChatPage */}
      <div className="hidden md:block">
        <Sidebar />
      </div>
      <div className="flex-1 overflow-hidden">{children}</div>
    </div>
  )
}
