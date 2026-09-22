package api

import (
	"html/template"
	"net/http"
	"time"

	"iconix-cron/internal/auth"
	"iconix-cron/internal/config"
	"iconix-cron/internal/repository"
	"iconix-cron/web"
)

type WebHandler struct {
	jobRepo        *repository.JobRepository
	logRepo        *repository.LogRepository
	tmpl           *template.Template
	cfg            *config.Config
	sessionManager *auth.SessionManager
}

func NewWebHandler(jobRepo *repository.JobRepository, logRepo *repository.LogRepository, cfg *config.Config, sessionManager *auth.SessionManager) (*WebHandler, error) {
	tmpl, err := template.ParseFS(web.TemplateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &WebHandler{
		jobRepo:        jobRepo,
		logRepo:        logRepo,
		tmpl:           tmpl,
		cfg:            cfg,
		sessionManager: sessionManager,
	}, nil
}

// AuthMiddleware protects web dashboard routes
func (h *WebHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil || !h.sessionManager.ValidateSession(cookie.Value) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *WebHandler) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if cookie, err := r.Cookie("session_token"); err == nil && h.sessionManager.ValidateSession(cookie.Value) {
		http.Redirect(w, r, "/web", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "login.html", map[string]interface{}{})
}

func (h *WebHandler) ProcessLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := r.FormValue("username")
	pass := r.FormValue("password")

	if user != h.cfg.AdminUsername || pass != h.cfg.AdminPassword {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = h.tmpl.ExecuteTemplate(w, "login.html", map[string]interface{}{
			"Error": "Username atau password salah.",
		})
		return
	}

	// Create session token (1 hour expiration)
	token, expiresAt := h.sessionManager.CreateSession(user)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/web", http.StatusSeeOther)
}

func (h *WebHandler) ProcessLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session_token"); err == nil {
		h.sessionManager.DeleteSession(cookie.Value)
	}

	// Expire cookie immediately
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	jobs, _, err := h.jobRepo.GetAll(1, 500, "id", "desc")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var activeCount int
	var totalExecutions, totalFailures int64
	var activeEPD, inactiveEPD, totalEPD int64

	for _, j := range jobs {
		epd := j.GetEPD()
		if j.Status == 1 {
			activeCount++
			activeEPD += epd
		} else {
			inactiveEPD += epd
		}
		totalEPD += epd
		totalExecutions += j.TotalSuccess + j.TotalFail
		totalFailures += j.TotalFail
	}

	data := map[string]interface{}{
		"Jobs":            jobs,
		"TotalJobs":       len(jobs),
		"ActiveJobs":      activeCount,
		"TotalExecutions": totalExecutions,
		"TotalFailures":   totalFailures,
		"ActiveEPD":       activeEPD,
		"InactiveEPD":     inactiveEPD,
		"TotalEPD":        totalEPD,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "index.html", data)
}

func (h *WebHandler) LogsView(w http.ResponseWriter, r *http.Request) {
	logs, _, err := h.logRepo.GetAllLogs(1, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Logs": logs,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "logs.html", data)
}
