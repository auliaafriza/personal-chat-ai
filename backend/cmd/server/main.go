package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/auliaafriza/personalgpt-backend/internal/config"
	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/eval"
	"github.com/auliaafriza/personalgpt-backend/internal/handler"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
	"github.com/auliaafriza/personalgpt-backend/internal/tools"
	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server fatal: %v", err)
	}
}

func run() error {
	// Config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log.Printf("→ Starting PersonalGPT backend (env=%s)…", cfg.Env)

	// Database
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Printf("✓ Database connected")

	// Repositories
	convRepo := db.NewConversationRepo(pool)
	msgRepo := db.NewMessageRepo(pool)
	userRepo := db.NewUserRepo(pool)
	docRepo := db.NewDocumentRepo(pool)
	taskRepo := db.NewTaskRepo(pool)
	memoryRepo := db.NewMemoryRepo(pool)
	traceRepo := db.NewTraceRepo(pool)
	evalRepo := db.NewEvalRepo(pool)

	// Services
	anthropicSvc := service.NewGroq(cfg.GroqAPIKey)
	embedder := service.NewEmbedder(cfg.VoyageAPIKey)
	reranker := service.NewReranker(cfg.VoyageAPIKey)
	retriever := service.NewRetriever(docRepo, embedder, reranker)
	translator := service.NewTranslator(anthropicSvc)

	// Workspace (Minggu 8) — per-user sandbox folder utk coding tools.
	ws, err := workspace.New(cfg.WorkspaceRoot)
	if err != nil {
		return err
	}
	log.Printf("✓ Workspace root: %s", ws.Root())

	// Tool registry (Minggu 7 + 8 + 9).
	toolReg := tools.NewRegistry(
		// Generic tools (Minggu 7)
		tools.NewCalculator(),
		tools.NewCurrentTime(),
		tools.NewFetchURL(),
		// Coding tools (Minggu 8)
		tools.NewReadFile(ws),
		tools.NewWriteFile(ws),
		tools.NewListDirectory(ws),
		tools.NewSearchCode(ws),
		tools.NewRunShell(ws),
		// Task tools (Minggu 9)
		tools.NewCreateTask(taskRepo),
		tools.NewListTasks(taskRepo),
		tools.NewCompleteTask(taskRepo),
		tools.NewDeleteTask(taskRepo),
		tools.NewRemindMe(taskRepo),
		// Google productivity tools (Minggu 9)
		tools.NewListCalendarEvents(),
		tools.NewCreateCalendarEvent(),
		tools.NewUpdateCalendarEvent(),
		tools.NewDeleteCalendarEvent(),
		tools.NewSearchGmail(),
		tools.NewReadGmailMessage(),
		// Memory tools (Minggu 10)
		tools.NewRememberThis(memoryRepo, embedder),
		tools.NewForgetMemory(memoryRepo),
		tools.NewUpdateMemoryTool(memoryRepo, embedder),
		// Translation tool
		tools.NewTranslate(translator),
	)
	if cfg.TavilyAPIKey != "" {
		toolReg.Register(tools.NewWebSearch(cfg.TavilyAPIKey))
		log.Printf("✓ web_search enabled (Tavily)")
	} else {
		log.Printf("⚠ TAVILY_API_KEY empty — web_search tool disabled")
	}
	log.Printf("✓ Tools registered: %v", toolReg.Names())

	// Handlers
	convH := handler.NewConversationHandler(convRepo)
	msgH := handler.NewMessageHandler(msgRepo, convRepo)
	titleH := handler.NewTitleHandler(convRepo, msgRepo, anthropicSvc)
	chatH := handler.NewChatHandler(convRepo, msgRepo, docRepo, memoryRepo, traceRepo, anthropicSvc, retriever, embedder, toolReg)
	meH := handler.NewMeHandler(userRepo)
	docH := handler.NewDocumentHandler(docRepo, embedder, retriever)
	taskH := handler.NewTaskHandler(taskRepo)
	memoryH := handler.NewMemoryHandler(memoryRepo, embedder)

	// Eval (Minggu 11) — retrieval eval reuses shared Retriever; judge pakai Groq small model.
	retrievalEv := eval.NewRetrievalEvaluator(retriever)
	judgeEv := eval.NewJudgeEvaluator(anthropicSvc)
	observabilityH := handler.NewObservabilityHandler(traceRepo)
	evalH := handler.NewEvalHandler(evalRepo, msgRepo, convRepo, retrievalEv, judgeEv)

	// Translation (on-demand endpoint)
	translateH := handler.NewTranslateHandler(translator)

	// Middleware
	authMw := appmw.Auth(cfg.AuthSecret, userRepo)

	// Router
	r := chi.NewRouter()

	r.Use(appmw.Logger)
	r.Use(appmw.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Vercel-Ai-Data-Stream"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Public route
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// Rate limiter untuk expensive endpoints (Minggu 12).
	// Capacity 20 burst, refill 0.5/sec = steady 1 req per 2 sec. Cukup lega
	// untuk chat interaktif + upload dokumen sesekali; block flood/abuse.
	expensiveLimiter := appmw.PerUser(20, 0.5)

	// Protected routes (require valid JWT)
	r.Group(func(r chi.Router) {
		r.Use(authMw)

		// Current user
		r.Get("/me", meH.Get)
		r.Put("/me/settings", meH.UpdateSettings)

		// Conversations
		r.Route("/conversations", func(r chi.Router) {
			r.Get("/", convH.List)
			r.Post("/", convH.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", convH.Get)
				r.Patch("/", convH.Update)
				r.Delete("/", convH.Delete)
				r.Get("/messages", msgH.List)
				r.Post("/title", titleH.Generate)
			})
		})

		// Streaming chat (rate-limited)
		r.With(expensiveLimiter.Middleware).Post("/chat", chatH.Stream)

		// Documents (Minggu 4) — upload + search rate-limited
		r.Route("/documents", func(r chi.Router) {
			r.Get("/", docH.List)
			r.With(expensiveLimiter.Middleware).Post("/", docH.Upload)
			r.With(expensiveLimiter.Middleware).Post("/search", docH.Search)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", docH.Get)
				r.Delete("/", docH.Delete)
			})
		})

		// Tasks (Minggu 9)
		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", taskH.List)
			r.Post("/", taskH.Create)
			r.Patch("/{id}", taskH.Update)
			r.Delete("/{id}", taskH.Delete)
		})

		// Memories (Minggu 10)
		r.Route("/memories", func(r chi.Router) {
			r.Get("/", memoryH.List)
			r.Post("/", memoryH.Create)
			r.Patch("/{id}", memoryH.Update)
			r.Delete("/{id}", memoryH.Delete)
		})

		// Observability + Evals (Minggu 11)
		r.Route("/observability", func(r chi.Router) {
			r.Get("/traces", observabilityH.ListTraces)
			r.Get("/metrics", observabilityH.Metrics)
		})
		r.Route("/eval-sets", func(r chi.Router) {
			r.Get("/", evalH.ListSets)
			r.Post("/", evalH.CreateSet)
			r.Delete("/{id}", evalH.DeleteSet)
		})
		r.Route("/eval-runs", func(r chi.Router) {
			r.Get("/", evalH.ListRuns)
			r.Post("/retrieval", evalH.RunRetrievalEval)
			r.Post("/judge", evalH.RunJudgeEval)
		})

		// Translation (on-demand, rate-limited)
		r.With(expensiveLimiter.Middleware).Post("/translate", translateH.Translate)
	})

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		// No WriteTimeout — streaming responses can be long
		IdleTimeout: 120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("→ Listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-quit:
		log.Printf("→ %s received, shutting down…", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Printf("✓ Server stopped")
	return nil
}
