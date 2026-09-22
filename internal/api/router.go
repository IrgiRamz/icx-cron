package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"iconix-cron/internal/auth"
	"iconix-cron/internal/config"
	"iconix-cron/internal/repository"
	"iconix-cron/internal/scheduler"
)

func NewRouter(jobRepo *repository.JobRepository, logRepo *repository.LogRepository, engine *scheduler.SchedulerEngine, cfg *config.Config, sessionManager *auth.SessionManager) (http.Handler, error) {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	easycronHandler := NewEasycronHandler(jobRepo, logRepo, engine, cfg.APIKey)
	webHandler, err := NewWebHandler(jobRepo, logRepo, cfg, sessionManager)
	if err != nil {
		return nil, err
	}

	// Public Auth & Documentation & Static Routes
	r.Get("/login", webHandler.ShowLoginPage)
	r.Post("/login", webHandler.ProcessLogin)
	r.Get("/logout", webHandler.ProcessLogout)
	r.Get("/docs", webHandler.DocsView)
	r.Get("/docs/openapi.json", webHandler.ServeOpenAPI)
	r.Get("/favicon.svg", webHandler.ServeFavicon)

	// Protected Web Dashboard Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/web", http.StatusFound)
	})

	r.Group(func(r chi.Router) {
		r.Use(webHandler.AuthMiddleware)

		r.Get("/web", webHandler.Dashboard)
		r.Get("/web/logs", webHandler.LogsView)
	})

	// Easycron REST API Routes
	r.Route("/v1", func(r chi.Router) {
		r.Use(easycronHandler.AuthMiddleware)

		r.Get("/users/me", easycronHandler.GetCurrentUser)

		r.Get("/cron-jobs", easycronHandler.ListCronJobs)
		r.Post("/cron-jobs", easycronHandler.CreateCronJob)
		r.Get("/cron-jobs/{id}", easycronHandler.GetCronJob)
		r.Patch("/cron-jobs/{id}", easycronHandler.UpdateCronJob)
		r.Delete("/cron-jobs/{id}", easycronHandler.DeleteCronJob)

		r.Post("/cron-jobs/{id}/enable", easycronHandler.EnableCronJob)
		r.Post("/cron-jobs/{id}/disable", easycronHandler.DisableCronJob)
		r.Post("/cron-jobs/{id}/trigger", easycronHandler.TriggerCronJob)
		r.Get("/cron-jobs/{id}/logs", easycronHandler.GetJobLogs)
		r.Get("/cron-jobs/{id}/executions", easycronHandler.GetJobExecutionsAndPredictions)
	})

	return r, nil
}
