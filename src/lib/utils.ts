import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

/**
 * Merge Tailwind utility classes safely (clsx + tailwind-merge).
 * Pakai ini di SEMUA composition class — never `+` concat (eDOT §9).
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}
