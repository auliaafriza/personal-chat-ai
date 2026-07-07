import {
  ArrowRight,
  Beaker,
  Brain,
  CheckSquare,
  FileText,
  Github,
  Languages,
  MessageSquare,
  Sparkles,
  Terminal,
} from "lucide-react"
import Link from "next/link"

/**
 * Public landing page (Minggu 12).
 *
 * Middleware redirect logged-in visitors ke /chat, jadi page ini cuma
 * ke-render untuk anonymous visitor. Server component — nggak butuh JS.
 */
export default function LandingPage() {
  return (
    <main className="min-h-dvh bg-background">
      <Header />
      <Hero />
      <FeaturesGrid />
      <TechStack />
      <FinalCTA />
      <Footer />
    </main>
  )
}

function Header() {
  return (
    <header className="mx-auto flex w-full max-w-5xl items-center justify-between px-4 py-4">
      <div className="flex items-center gap-2 text-sm font-semibold">
        <Sparkles className="h-4 w-4 text-primary" />
        Personal Chat AI by Aulia
      </div>
      <div className="flex items-center gap-3 text-sm">
        <a
          href="https://github.com/auliaafriza"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1 text-muted-foreground transition-colors hover:text-foreground"
        >
          <Github className="h-4 w-4" /> GitHub
        </a>
        <Link
          href="/signin"
          className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-opacity hover:opacity-90"
        >
          Sign in
        </Link>
      </div>
    </header>
  )
}

function Hero() {
  return (
    <section className="mx-auto w-full max-w-4xl px-4 py-16 text-center md:py-24">
      <div className="mx-auto mb-4 inline-flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1 text-xs text-muted-foreground">
        <Sparkles className="h-3 w-3" /> 12-week AI Engineer roadmap · portfolio project
      </div>
      <h1 className="text-3xl font-semibold tracking-tight md:text-5xl">
        Chat AI pribadi dengan{" "}
        <span className="bg-gradient-to-r from-primary to-purple-400 bg-clip-text text-transparent">
          RAG, tools & memory
        </span>
      </h1>
      <p className="mx-auto mt-4 max-w-2xl text-sm text-muted-foreground md:text-base">
        Next.js 15 + Go + Neon Postgres + pgvector. Dokumen kamu di-embed dan
        di-retrieve dengan hybrid search + reranking. 24 tools untuk web
        search, coding assist, calendar, gmail, tasks, dan lebih.
      </p>
      <div className="mt-8 flex items-center justify-center gap-3">
        <Link
          href="/signin"
          className="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90"
        >
          Sign in with Google <ArrowRight className="h-3.5 w-3.5" />
        </Link>
        <a
          href="https://github.com/auliaafriza"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1.5 rounded-md border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
        >
          <Github className="h-3.5 w-3.5" /> Source code
        </a>
      </div>
    </section>
  )
}

const features = [
  {
    Icon: MessageSquare,
    title: "Streaming chat with tools",
    body:
      "Multi-turn function calling: web_search (Tavily), fetch_url (HTML→markdown), calculator, get_current_time, dan lainnya.",
  },
  {
    Icon: FileText,
    title: "RAG dokumen",
    body:
      "Upload txt/md/pdf/docx atau paste text. Voyage AI embeddings + pgvector HNSW. Hybrid search (vector + BM25 RRF) + rerank-2.",
  },
  {
    Icon: Brain,
    title: "Long-term memory",
    body:
      "LLM ingat fakta tentang kamu dengan tool remember_this. Auto-inject di setiap chat untuk personalisasi.",
  },
  {
    Icon: Terminal,
    title: "Coding assistant",
    body:
      "Per-user sandboxed workspace. Read/write file, search_code (regex), run_shell (allowlist read-only). Syntax highlight + diff viewer.",
  },
  {
    Icon: CheckSquare,
    title: "Productivity tools",
    body:
      "Google Calendar (list/create/update/delete), Gmail read-only search, tasks CRUD, reminders. OAuth scopes via Auth.js v5.",
  },
  {
    Icon: Languages,
    title: "Translate ID ↔ EN",
    body:
      "Klik tombol Translate di setiap response untuk toggle antara Bahasa Indonesia dan English. Atau minta LLM via chat.",
  },
  {
    Icon: Beaker,
    title: "Evals + observability",
    body:
      "Timing per stage + tokens + tool usage di-track per request. Retrieval eval (recall@k + MRR) + LLM-as-judge scoring.",
  },
]

function FeaturesGrid() {
  return (
    <section className="mx-auto w-full max-w-5xl px-4 py-12">
      <h2 className="text-center text-xl font-semibold md:text-2xl">Fitur</h2>
      <p className="mx-auto mt-1 max-w-xl text-center text-sm text-muted-foreground">
        7 kategori feature yang di-ship dalam 12 minggu — dari basic chat sampai observability.
      </p>
      <div className="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {features.map((f) => (
          <div key={f.title} className="rounded-lg border border-border bg-card p-4">
            <f.Icon className="mb-2 h-4 w-4 text-primary" />
            <h3 className="text-sm font-medium">{f.title}</h3>
            <p className="mt-1 text-xs text-muted-foreground">{f.body}</p>
          </div>
        ))}
      </div>
    </section>
  )
}

const techStack = [
  { name: "Next.js 15", role: "App Router + Server Actions" },
  { name: "Go 1.25", role: "chi router + pgx" },
  { name: "Neon Postgres", role: "pgvector + tsvector" },
  { name: "Auth.js v5", role: "Google OAuth + HS256 JWT" },
  { name: "Groq", role: "Llama chat + function calling" },
  { name: "Voyage AI", role: "Embeddings + rerank-2" },
  { name: "Tavily", role: "Web search API" },
  { name: "TanStack Query", role: "State + cache" },
  { name: "Tailwind + Radix", role: "UI primitives" },
]

function TechStack() {
  return (
    <section className="mx-auto w-full max-w-4xl px-4 py-12">
      <h2 className="text-center text-xl font-semibold md:text-2xl">Stack</h2>
      <div className="mt-6 flex flex-wrap justify-center gap-2 text-xs">
        {techStack.map((t) => (
          <div key={t.name} className="rounded-full border border-border bg-card px-3 py-1.5">
            <span className="font-medium">{t.name}</span>
            <span className="ml-1.5 text-muted-foreground">{t.role}</span>
          </div>
        ))}
      </div>
    </section>
  )
}

function FinalCTA() {
  return (
    <section className="mx-auto w-full max-w-3xl px-4 py-16 text-center">
      <h2 className="text-xl font-semibold md:text-2xl">Coba sendiri</h2>
      <p className="mx-auto mt-2 max-w-md text-sm text-muted-foreground">
        Sign in dengan Google account kamu. Chat history + dokumen + memory + tasks disimpan di user kamu sendiri.
      </p>
      <Link
        href="/signin"
        className="mt-6 inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90"
      >
        Sign in with Google <ArrowRight className="h-3.5 w-3.5" />
      </Link>
    </section>
  )
}

function Footer() {
  return (
    <footer className="mx-auto w-full max-w-5xl border-t border-border px-4 py-6 text-center text-xs text-muted-foreground">
      Built by Aulia Afriza · 12-week AI Engineer roadmap · 2026
    </footer>
  )
}
