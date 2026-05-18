import type { AxiosError, AxiosResponse, InternalAxiosRequestConfig } from "axios"
import axios from "axios"

/**
 * Single axios instance untuk Go BE (eDOT §6 — one axios instance per backend service).
 *
 * Kalau ada backend lain (misal apiCohere untuk reranking di Minggu 6),
 * bikin file terpisah di sini.
 */

const baseURL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"

const apiApp = axios.create({
  baseURL,
  timeout: 30_000,
  timeoutErrorMessage: "Request timed out. Server tidak merespon dalam 30 detik.",
})

apiApp.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (process.env.NODE_ENV !== "production") {
      console.warn(`[apiApp] → ${config.method?.toUpperCase()} ${config.url}`)
    }
    return config
  },
  (error: AxiosError) => Promise.reject(error),
)

apiApp.interceptors.response.use(
  (response: AxiosResponse) => response.data,
  (error: AxiosError) => {
    const status = error.response?.status ?? 0
    console.error(`[apiApp] ✗ ${error.config?.url} → ${status}`, error.message)
    return Promise.reject(error)
  },
)

export { apiApp, baseURL as apiBaseURL }
