import type { ReactNode } from "react"

import { Sidebar } from "@/components/layout/Sidebar"

export default function ChatLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-dvh bg-background">
      <Sidebar />
      <div className="flex-1 overflow-hidden">{children}</div>
    </div>
  )
}
