"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { ArrowLeft, Loader2, Monitor, Moon, Sun } from "lucide-react"
import Link from "next/link"
import { useTheme } from "next-themes"
import { useEffect } from "react"
import { Controller, useForm } from "react-hook-form"

import { cn } from "@/lib/utils"

import {
  AVAILABLE_MODELS,
  DEFAULT_MODEL,
  DEFAULT_TEMPERATURE,
  MAX_SYSTEM_PROMPT_LENGTH,
  MAX_TEMPERATURE,
  MIN_TEMPERATURE,
} from "../constants"
import { useGetMe } from "../services/me/get"
import { useMutationUpdateSettings } from "../services/me/put"
import { settingsFormSchema, type SettingsFormValues } from "../types"

export function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const { data: user, isLoading } = useGetMe()
  const updateMutation = useMutationUpdateSettings()

  const form = useForm<SettingsFormValues>({
    resolver: zodResolver(settingsFormSchema),
    mode: "onChange",
    defaultValues: {
      defaultModel: DEFAULT_MODEL,
      defaultTemperature: DEFAULT_TEMPERATURE,
      systemPrompt: "",
    },
  })

  // Reset form once user data loads
  useEffect(() => {
    if (user) {
      form.reset({
        defaultModel: user.defaultModel,
        defaultTemperature: user.defaultTemperature,
        systemPrompt: user.systemPrompt,
      })
    }
  }, [user, form])

  const onSubmit = (values: SettingsFormValues) => {
    updateMutation.mutate(values)
  }

  const promptLength = form.watch("systemPrompt")?.length ?? 0
  const isOverLimit = promptLength > MAX_SYSTEM_PROMPT_LENGTH

  return (
    <main className="flex h-dvh flex-col bg-background">
      <header className="flex items-center gap-3 border-b border-border px-4 py-3">
        <Link
          href="/chat"
          className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent"
          aria-label="Back to chat"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <h1 className="text-base font-semibold">Settings</h1>
      </header>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-2xl px-4 py-8">
          {isLoading ? (
            <div className="flex justify-center py-12 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : (
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
              {/* Profile section */}
              <section className="space-y-2 rounded-lg border border-border bg-card p-4">
                <h2 className="text-sm font-medium">Akun</h2>
                <div className="flex items-center gap-3 pt-2">
                  {user?.avatarUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={user.avatarUrl}
                      alt={user.name}
                      className="h-12 w-12 rounded-full object-cover"
                    />
                  ) : (
                    <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary text-base font-semibold text-primary-foreground">
                      {(user?.name ?? user?.email ?? "?").charAt(0).toUpperCase()}
                    </div>
                  )}
                  <div>
                    <p className="text-sm font-medium">{user?.name ?? "User"}</p>
                    <p className="text-xs text-muted-foreground">{user?.email}</p>
                  </div>
                </div>
              </section>

              {/* Theme picker */}
              <section className="space-y-3 rounded-lg border border-border bg-card p-4">
                <div>
                  <h2 className="text-sm font-medium">Tampilan</h2>
                  <p className="text-xs text-muted-foreground">Pilih tema warna antarmuka.</p>
                </div>
                <div className="grid grid-cols-3 gap-2">
                  {[
                    { value: "light", label: "Light", Icon: Sun },
                    { value: "dark", label: "Dark", Icon: Moon },
                    { value: "system", label: "System", Icon: Monitor },
                  ].map(({ value, label, Icon }) => (
                    <button
                      key={value}
                      type="button"
                      onClick={() => setTheme(value)}
                      className={cn(
                        "flex flex-col items-center gap-2 rounded-md border border-border p-3 text-sm transition-colors",
                        theme === value ? "border-primary bg-accent/30" : "hover:bg-accent/50",
                      )}
                    >
                      <Icon className="h-4 w-4" />
                      {label}
                    </button>
                  ))}
                </div>
              </section>

              {/* Model picker */}
              <section className="space-y-3 rounded-lg border border-border bg-card p-4">
                <div>
                  <h2 className="text-sm font-medium">Default model</h2>
                  <p className="text-xs text-muted-foreground">
                    Dipakai untuk conversation baru. Bisa di-override per-conversation.
                  </p>
                </div>
                <Controller
                  control={form.control}
                  name="defaultModel"
                  render={({ field }) => (
                    <div className="space-y-2">
                      {AVAILABLE_MODELS.map((model) => (
                        <label
                          key={model.id}
                          className={cn(
                            "flex cursor-pointer items-start gap-3 rounded-md border border-border p-3 text-sm transition-colors",
                            field.value === model.id ? "border-primary bg-accent/30" : "hover:bg-accent/50",
                          )}
                        >
                          <input
                            type="radio"
                            value={model.id}
                            checked={field.value === model.id}
                            onChange={() => field.onChange(model.id)}
                            className="mt-1"
                          />
                          <div>
                            <p className="font-medium">{model.label}</p>
                            <p className="text-xs text-muted-foreground">{model.description}</p>
                          </div>
                        </label>
                      ))}
                      {form.formState.errors.defaultModel ? (
                        <p className="text-xs text-destructive">
                          {form.formState.errors.defaultModel.message}
                        </p>
                      ) : null}
                    </div>
                  )}
                />
              </section>

              {/* Temperature slider */}
              <section className="space-y-3 rounded-lg border border-border bg-card p-4">
                <div>
                  <h2 className="text-sm font-medium">Temperature</h2>
                  <p className="text-xs text-muted-foreground">
                    0 = deterministik, 2 = sangat kreatif (dan kadang ngawur). Default: 0.7.
                  </p>
                </div>
                <Controller
                  control={form.control}
                  name="defaultTemperature"
                  render={({ field }) => (
                    <div className="space-y-2">
                      <div className="flex items-center gap-4">
                        <input
                          type="range"
                          min={MIN_TEMPERATURE}
                          max={MAX_TEMPERATURE}
                          step={0.1}
                          value={field.value}
                          onChange={(e) => field.onChange(parseFloat(e.target.value))}
                          className="flex-1 accent-primary"
                        />
                        <span className="w-10 text-right font-mono text-sm">
                          {field.value.toFixed(1)}
                        </span>
                      </div>
                      {form.formState.errors.defaultTemperature ? (
                        <p className="text-xs text-destructive">
                          {form.formState.errors.defaultTemperature.message}
                        </p>
                      ) : null}
                    </div>
                  )}
                />
              </section>

              {/* System prompt */}
              <section className="space-y-3 rounded-lg border border-border bg-card p-4">
                <div>
                  <h2 className="text-sm font-medium">Custom system prompt</h2>
                  <p className="text-xs text-muted-foreground">
                    Opsional. Kosongkan untuk pakai default prompt.
                  </p>
                </div>
                <textarea
                  {...form.register("systemPrompt")}
                  rows={6}
                  placeholder="Contoh: Kamu adalah Personal Chat AI by Aulia — selalu jawab dalam bahasa Indonesia, gunakan emoji minimal."
                  className={cn(
                    "w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm",
                    "focus:outline-none focus:ring-2 focus:ring-ring",
                    isOverLimit && "border-destructive focus:ring-destructive",
                  )}
                />
                <div className="flex justify-between text-xs text-muted-foreground">
                  {form.formState.errors.systemPrompt ? (
                    <span className="text-destructive">
                      {form.formState.errors.systemPrompt.message}
                    </span>
                  ) : (
                    <span />
                  )}
                  <span className={cn(isOverLimit && "text-destructive")}>
                    {promptLength} / {MAX_SYSTEM_PROMPT_LENGTH}
                  </span>
                </div>
              </section>

              <div className="flex justify-end gap-2 pt-2">
                <Link
                  href="/chat"
                  className="rounded-lg border border-border bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
                >
                  Cancel
                </Link>
                <button
                  type="submit"
                  disabled={
                    !form.formState.isDirty ||
                    !form.formState.isValid ||
                    updateMutation.isPending
                  }
                  className={cn(
                    "flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity",
                    "hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40",
                  )}
                >
                  {updateMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                  Save changes
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </main>
  )
}
