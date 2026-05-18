"use client"

import type { ReactNode } from "react"
import { useState } from "react"

import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ReactQueryDevtools } from "@tanstack/react-query-devtools"

/**
 * Singleton-per-app QueryClient (eDOT §6).
 * Mount sekali di root layout — never per-page, atau cache hilang antar route.
 */
export function QueryProvider({ children }: { children: ReactNode }) {
  // useState ensures the client is created once per browser session, not per render
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60_000, // 1 minute
            refetchOnWindowFocus: false,
            retry: 1,
          },
        },
      }),
  )

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}
