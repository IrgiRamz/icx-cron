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

	"iconix-cron/internal/api"
	"iconix-cron/internal/auth"
	"iconix-cron/internal/config"
	"iconix-cron/internal/db"
	"iconix-cron/internal/repository"
	"iconix-cron/internal/scheduler"
)

func main() {
	log.Println("================================================")
	log.Println(" Starting Iconix Cron Engine (Self-Hosted) ")
	log.Println("================================================")

	cfg := config.LoadConfig()

	// Session Manager (1 Hour Session Expiration)
	sessionManager := auth.NewSessionManager(1 * time.Hour)

	// Initialize DB
	database, err := db.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Fatal: Failed to initialize DB: %v", err)
	}
	defer database.Close()

	// Repositories
	jobRepo := repository.NewJobRepository(database)
	logRepo := repository.NewLogRepository(database)

	// Executor & Scheduler Engine
	executor := scheduler.NewExecutor(jobRepo, logRepo, cfg.MaxWorkers)
	engine := scheduler.NewSchedulerEngine(jobRepo, logRepo, executor, cfg.LogRetentionDays)

	if err := engine.Start(); err != nil {
		log.Fatalf("Fatal: Failed to start scheduler engine: %v", err)
	}
	defer engine.Stop()

	// HTTP Router
	router, err := api.NewRouter(jobRepo, logRepo, engine, cfg, sessionManager)
	if err != nil {
		log.Fatalf("Fatal: Failed to set up HTTP router: %v", err)
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("[SERVER] HTTP Server listening on port %s", cfg.Port)
		log.Printf("[SERVER] Dashboard available at http://localhost:%s/web", cfg.Port)
		log.Printf("[SERVER] Easycron API ready at http://localhost:%s/v1/cron-jobs", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal: HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[SERVER] Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[SERVER] Server shutdown forced: %v", err)
	}

	log.Println("[SERVER] Iconix Cron Engine stopped successfully.")
}
