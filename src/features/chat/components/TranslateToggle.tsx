"use client"

import { Languages, Loader2 } from "lucide-react"
import { useState } from "react"

import { cn } from "@/lib/utils"

import { type Lang, useMutationTranslate } from "../services/translate/post"

interface TranslateToggleProps {
  content: string
  onSwapContent: (newContent: string | null) => void
  showingTranslation: boolean
}

/**
 * Tombol translate — auto-detect language current dan translate ke opposite.
 * State ada di parent (ChatBubble), tombol ini cuma trigger + toggle.
 */
export function TranslateToggle({ content, onSwapContent, showingTranslation }: TranslateToggleProps) {
  const translateMut = useMutationTranslate()
  const [cache, setCache] = useState<string | null>(null)

  const handleClick = () => {
    if (showingTranslation) {
      // Toggle back to original
      onSwapContent(null)
      return
    }
    if (cache) {
      // Second click after fresh translate — reuse cache
      onSwapContent(cache)
      return
    }
    // Guess target language: kalau content mengandung banyak kata Indonesia → EN, else → ID.
    const target: Lang = looksIndonesian(content) ? "en" : "id"
    translateMut.mutate(
      { text: content, target },
      {
        onSuccess: (res) => {
          setCache(res.translated)
          onSwapContent(res.translated)
        },
      },
    )
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={translateMut.isPending}
      title={showingTranslation ? "Tampilkan asli" : "Translate"}
      className={cn(
        "flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] text-muted-foreground transition-colors",
        "hover:bg-background/60 hover:text-foreground",
        showingTranslation && "bg-background/60 text-foreground",
        translateMut.isPending && "opacity-50",
      )}
    >
      {translateMut.isPending ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : (
        <Languages className="h-3 w-3" />
      )}
      {showingTranslation ? "Original" : "Translate"}
    </button>
  )
}

// Heuristik sederhana — count occurrence dari common Indonesian markers.
// Nggak perfect tapi cukup untuk decide target language default.
function looksIndonesian(text: string): boolean {
  const s = " " + text.toLowerCase() + " "
  const idMarkers = [
    " yang ", " dan ", " atau ", " untuk ", " dengan ", " di ", " ke ", " dari ",
    " ini ", " itu ", " adalah ", " tidak ", " juga ", " kalau ", " karena ",
    " saya ", " kamu ", " aku ", " nya ", " nggak ", " bisa ",
  ]
  let hits = 0
  for (const m of idMarkers) {
    if (s.includes(m)) hits++
  }
  return hits >= 2
}
