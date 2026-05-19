"use client"

import { useEffect } from "react"

interface UseChatShortcutsOptions {
  /** Cmd/Ctrl + K — start a new chat. */
  onNewChat?: () => void
  /** Cmd/Ctrl + / — focus the input field. */
  onFocusInput?: () => void
}

/**
 * Attach keyboard shortcuts at window level. Auto-cleans on unmount.
 * Ignores keystrokes inside text inputs untuk Cmd+K only (Cmd+/ tetap aktif).
 */
export function useChatShortcuts({ onNewChat, onFocusInput }: UseChatShortcutsOptions) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const isMeta = e.metaKey || e.ctrlKey

      if (isMeta && (e.key === "k" || e.key === "K")) {
        e.preventDefault()
        onNewChat?.()
        return
      }

      if (isMeta && e.key === "/") {
        e.preventDefault()
        onFocusInput?.()
      }
    }

    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [onNewChat, onFocusInput])
}
