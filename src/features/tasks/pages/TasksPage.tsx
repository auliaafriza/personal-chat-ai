"use client"

import dayjs from "dayjs"
import relativeTime from "dayjs/plugin/relativeTime"
dayjs.extend(relativeTime)
import {
  ArrowLeft,
  Bell,
  CheckCircle2,
  Circle,
  Clock,
  Loader2,
  Plus,
  Trash2,
} from "lucide-react"
import Link from "next/link"
import { useState } from "react"

import { cn } from "@/lib/utils"

import { useMutationCreateTask } from "../services/post"
import { useMutationDeleteTask } from "../services/delete"
import { useGetTasks } from "../services/list/get"
import { useMutationUpdateTask } from "../services/patch"
import type { Task, TaskDueFilter, TaskStatusFilter } from "../types/api"

const DUE_FILTERS: { value: TaskDueFilter; label: string }[] = [
  { value: "all", label: "Semua" },
  { value: "overdue", label: "Telat" },
  { value: "today", label: "Hari ini" },
  { value: "upcoming", label: "Datang" },
  { value: "no_due", label: "Tanpa due" },
]

const STATUS_FILTERS: { value: TaskStatusFilter; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "completed", label: "Done" },
  { value: "all", label: "Semua" },
]

export function TasksPage() {
  const [status, setStatus] = useState<TaskStatusFilter>("pending")
  const [due, setDue] = useState<TaskDueFilter>("all")
  const { data: tasks, isLoading } = useGetTasks({ status, due })

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
        <h1 className="text-base font-semibold">Tasks</h1>
      </header>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-2xl space-y-4 px-4 py-6">
          <NewTaskForm />

          <div className="flex flex-wrap gap-2 text-xs">
            <FilterGroup label="Status" value={status} options={STATUS_FILTERS} onChange={setStatus} />
            <FilterGroup label="Due" value={due} options={DUE_FILTERS} onChange={setDue} />
          </div>

          {isLoading ? (
            <div className="flex justify-center py-8 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : !tasks || tasks.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
              Nggak ada task di filter ini.
            </div>
          ) : (
            <ul className="space-y-2">
              {tasks.map((task) => (
                <TaskItem key={task.id} task={task} />
              ))}
            </ul>
          )}
        </div>
      </div>
    </main>
  )
}

function NewTaskForm() {
  const createMut = useMutationCreateTask()
  const [title, setTitle] = useState("")
  const [dueDate, setDueDate] = useState("")

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = title.trim()
    if (!trimmed) return
    createMut.mutate(
      {
        title: trimmed,
        dueDate: dueDate ? new Date(dueDate).toISOString() : undefined,
      },
      {
        onSuccess: () => {
          setTitle("")
          setDueDate("")
        },
      },
    )
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-2 rounded-lg border border-border bg-card p-3 sm:flex-row sm:items-center"
    >
      <input
        type="text"
        placeholder="Tambah task baru…"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
      />
      <input
        type="datetime-local"
        value={dueDate}
        onChange={(e) => setDueDate(e.target.value)}
        className="rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring sm:w-auto"
        title="Due date (opsional)"
      />
      <button
        type="submit"
        disabled={!title.trim() || createMut.isPending}
        className={cn(
          "flex items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground transition-opacity",
          "hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40",
        )}
      >
        {createMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
        Add
      </button>
    </form>
  )
}

function TaskItem({ task }: { task: Task }) {
  const updateMut = useMutationUpdateTask()
  const deleteMut = useMutationDeleteTask()

  const dueLabel = task.dueDate ? formatDue(task.dueDate, task.completed) : null

  return (
    <li
      className={cn(
        "flex items-start gap-3 rounded-lg border border-border bg-card p-3 transition-colors",
        task.completed && "opacity-60",
      )}
    >
      <button
        type="button"
        onClick={() => updateMut.mutate({ id: task.id, completed: !task.completed })}
        disabled={updateMut.isPending}
        className="mt-0.5 shrink-0 text-muted-foreground transition-colors hover:text-foreground"
        aria-label={task.completed ? "Mark pending" : "Mark done"}
      >
        {task.completed ? (
          <CheckCircle2 className="h-5 w-5 text-emerald-500" />
        ) : (
          <Circle className="h-5 w-5" />
        )}
      </button>

      <div className="min-w-0 flex-1">
        <p className={cn("text-sm font-medium", task.completed && "line-through")}>
          {task.title}
        </p>
        {task.description ? (
          <p className="mt-0.5 text-xs text-muted-foreground">{task.description}</p>
        ) : null}
        <div className="mt-1 flex items-center gap-2 text-[11px] text-muted-foreground">
          {task.isReminder ? (
            <span className="inline-flex items-center gap-1 text-amber-500">
              <Bell className="h-3 w-3" /> Reminder
            </span>
          ) : null}
          {dueLabel ? (
            <span className={cn("inline-flex items-center gap-1", dueLabel.kind)}>
              <Clock className="h-3 w-3" /> {dueLabel.text}
            </span>
          ) : null}
        </div>
      </div>

      <button
        type="button"
        onClick={() => {
          if (window.confirm(`Hapus "${task.title}"?`)) {
            deleteMut.mutate(task.id)
          }
        }}
        disabled={deleteMut.isPending}
        className="shrink-0 text-muted-foreground transition-colors hover:text-destructive"
        aria-label="Delete"
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </li>
  )
}

function FilterGroup<T extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: T
  options: { value: T; label: string }[]
  onChange: (v: T) => void
}) {
  return (
    <div className="flex items-center gap-1">
      <span className="text-muted-foreground">{label}:</span>
      <div className="flex rounded-md border border-border bg-card">
        {options.map((opt) => (
          <button
            key={opt.value}
            type="button"
            onClick={() => onChange(opt.value)}
            className={cn(
              "px-2 py-1 transition-colors first:rounded-l-md last:rounded-r-md",
              value === opt.value ? "bg-primary text-primary-foreground" : "hover:bg-accent",
            )}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </div>
  )
}

function formatDue(due: string, completed: boolean): { text: string; kind: string } {
  const d = dayjs(due)
  const now = dayjs()
  if (completed) {
    return { text: d.format("DD MMM HH:mm"), kind: "" }
  }
  if (d.isBefore(now)) {
    return { text: "Telat " + d?.fromNow() ? d.format("DD MMM HH:mm") : d.format("DD MMM"), kind: "text-destructive" }
  }
  if (d.isSame(now, "day")) {
    return { text: "Hari ini " + d.format("HH:mm"), kind: "text-amber-500" }
  }
  return { text: d.format("DD MMM HH:mm"), kind: "" }
}
