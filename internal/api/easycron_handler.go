package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"iconix-cron/internal/auth"
	"iconix-cron/internal/model"
	"iconix-cron/internal/repository"
	"iconix-cron/internal/scheduler"
)

type EasycronHandler struct {
	jobRepo        *repository.JobRepository
	logRepo        *repository.LogRepository
	engine         *scheduler.SchedulerEngine
	apiKey         string
	sessionManager *auth.SessionManager
}

func NewEasycronHandler(jobRepo *repository.JobRepository, logRepo *repository.LogRepository, engine *scheduler.SchedulerEngine, apiKey string, sessionManager *auth.SessionManager) *EasycronHandler {
	return &EasycronHandler{
		jobRepo:        jobRepo,
		logRepo:        logRepo,
		engine:         engine,
		apiKey:         apiKey,
		sessionManager: sessionManager,
	}
}

// AuthMiddleware validates Session Cookie OR X-API-Key header / token query parameter
func (h *EasycronHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check Session Cookie (for web dashboard requests)
		if cookie, err := r.Cookie("session_token"); err == nil && cookie.Value != "" {
			if h.sessionManager != nil && h.sessionManager.ValidateSession(cookie.Value) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 2. Check X-API-Key / token / api_key query parameter (for external API clients)
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("token")
		}
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}

		if h.apiKey != "" && key == h.apiKey {
			next.ServeHTTP(w, r)
			return
		}

		h.respondError(w, http.StatusUnauthorized, "Unauthorized: Invalid Session or API Key")
	})
}

func (h *EasycronHandler) respondJSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	if _, exists := r.URL.Query()["pretty"]; exists {
		encoder.SetIndent("", "  ")
	}
	_ = encoder.Encode(data)
}

func (h *EasycronHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// GET /v1/users/me
func (h *EasycronHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"email_address":      "admin@iconix.co.id",
		"email_subscription": true,
		"timezone":           "Asia/Jakarta",
		"user_id":            1,
	}
	h.respondJSON(w, r, http.StatusOK, resp)
}

// GET /v1/cron-jobs
func (h *EasycronHandler) ListCronJobs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 10
	}

	sortBy := r.URL.Query().Get("sort_by")
	order := r.URL.Query().Get("order")

	jobs, total, err := h.jobRepo.GetAll(page, pageSize, sortBy, order)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var cronJobsResp []model.EasycronJobResponse
	for _, j := range jobs {
		cronJobsResp = append(cronJobsResp, j.ToEasycronResponse())
	}
	if cronJobsResp == nil {
		cronJobsResp = []model.EasycronJobResponse{}
	}

	meta := map[string]interface{}{
		"page":         page,
		"page_size":    pageSize,
		"result_count": len(cronJobsResp),
		"total_count":  total,
	}

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"cron_jobs": cronJobsResp,
		"meta":      meta,
	})
}

// GET /v1/cron-jobs/{id}
func (h *EasycronHandler) GetCronJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid cron job ID")
		return
	}

	job, err := h.jobRepo.GetByID(id)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job == nil {
		h.respondError(w, http.StatusNotFound, "Cron job not found")
		return
	}

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"cron_job": job.ToEasycronResponse(),
	})
}

// POST /v1/cron-jobs
func (h *EasycronHandler) CreateCronJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name           string `json:"cron_job_name"`
		URL            string `json:"url"`
		Method         string `json:"http_method"`
		CronExpression string `json:"cron_expression"`
		Timeout        int    `json:"timeout"`
		Status         int    `json:"status"`
	}

	// Try reading JSON body first, or fallback to Form values
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&input)
	} else {
		_ = r.ParseForm()
		input.Name = r.FormValue("cron_job_name")
		input.URL = r.FormValue("url")
		input.Method = r.FormValue("http_method")
		input.CronExpression = r.FormValue("cron_expression")
		input.Timeout, _ = strconv.Atoi(r.FormValue("timeout"))
		if r.FormValue("status") != "" {
			input.Status, _ = strconv.Atoi(r.FormValue("status"))
		} else {
			input.Status = 1
		}
	}

	if input.URL == "" || input.CronExpression == "" {
		h.respondError(w, http.StatusBadRequest, "URL and cron_expression are required")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = "Unnamed"
	}
	if input.Method == "" {
		input.Method = "GET"
	}
	if input.Timeout <= 0 {
		input.Timeout = 10
	}

	job := &model.Job{
		Name:           input.Name,
		URL:            input.URL,
		Method:         strings.ToUpper(input.Method),
		CronExpression: input.CronExpression,
		Status:         input.Status,
		TimeoutSeconds: input.Timeout,
	}

	if err := h.jobRepo.Create(job); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = h.engine.ReloadJob(job.ID)

	h.respondJSON(w, r, http.StatusCreated, map[string]interface{}{
		"cron_job_id": job.ID,
		"cron_job":    job.ToEasycronResponse(),
	})
}

// PATCH /v1/cron-jobs/{id}
func (h *EasycronHandler) UpdateCronJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid cron job ID")
		return
	}

	existing, err := h.jobRepo.GetByID(id)
	if err != nil || existing == nil {
		h.respondError(w, http.StatusNotFound, "Cron job not found")
		return
	}

	var input struct {
		Name           string `json:"cron_job_name"`
		URL            string `json:"url"`
		Method         string `json:"http_method"`
		CronExpression string `json:"cron_expression"`
		Timeout        int    `json:"timeout"`
		Status         *int   `json:"status"`
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&input)
	} else {
		_ = r.ParseForm()
		if name := r.FormValue("cron_job_name"); name != "" {
			input.Name = name
		}
		if urlStr := r.FormValue("url"); urlStr != "" {
			input.URL = urlStr
		}
		if method := r.FormValue("http_method"); method != "" {
			input.Method = method
		}
		if expr := r.FormValue("cron_expression"); expr != "" {
			input.CronExpression = expr
		}
		if tStr := r.FormValue("timeout"); tStr != "" {
			input.Timeout, _ = strconv.Atoi(tStr)
		}
		if sStr := r.FormValue("status"); sStr != "" {
			s, _ := strconv.Atoi(sStr)
			input.Status = &s
		}
	}

	if input.Name != "" {
		existing.Name = strings.TrimSpace(input.Name)
		if existing.Name == "" {
			existing.Name = "Unnamed"
		}
	}
	if input.URL != "" {
		existing.URL = input.URL
	}
	if input.Method != "" {
		existing.Method = strings.ToUpper(input.Method)
	}
	if input.CronExpression != "" {
		existing.CronExpression = input.CronExpression
	}
	if input.Timeout > 0 {
		existing.TimeoutSeconds = input.Timeout
	}
	if input.Status != nil {
		existing.Status = *input.Status
	}

	if err := h.jobRepo.Update(existing); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = h.engine.ReloadJob(existing.ID)

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"cron_job_id": existing.ID,
		"cron_job":    existing.ToEasycronResponse(),
	})
}

// POST /v1/cron-jobs/{id}/enable
func (h *EasycronHandler) EnableCronJob(w http.ResponseWriter, r *http.Request) {
	h.setJobStatus(w, r, 1)
}

// POST /v1/cron-jobs/{id}/disable
func (h *EasycronHandler) DisableCronJob(w http.ResponseWriter, r *http.Request) {
	h.setJobStatus(w, r, 0)
}

func (h *EasycronHandler) setJobStatus(w http.ResponseWriter, r *http.Request, status int) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid cron job ID")
		return
	}

	if err := h.jobRepo.UpdateStatus(id, status); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = h.engine.ReloadJob(id)

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"cron_job_id": id,
	})
}

// POST /v1/cron-jobs/{id}/trigger
func (h *EasycronHandler) TriggerCronJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid cron job ID")
		return
	}

	if err := h.engine.TriggerJobNow(id); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"message":     "Job triggered successfully",
		"cron_job_id": id,
	})
}

// DELETE /v1/cron-jobs/{id}
func (h *EasycronHandler) DeleteCronJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid cron job ID")
		return
	}

	h.engine.RemoveJob(id)

	if err := h.jobRepo.Delete(id); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"cron_job_id": id,
	})
}

// GET /v1/cron-jobs/{id}/logs
func (h *EasycronHandler) GetJobLogs(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid cron job ID")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	logs, total, err := h.logRepo.GetByJobID(id, page, pageSize)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"logs":        logs,
		"total_count": total,
	})
}

// GET /v1/cron-jobs/{id}/executions
func (h *EasycronHandler) GetJobExecutionsAndPredictions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid cron job ID")
		return
	}

	job, err := h.jobRepo.GetByID(id)
	if err != nil || job == nil {
		h.respondError(w, http.StatusNotFound, "Cron job not found")
		return
	}

	// 1. Fetch 10 latest execution logs
	logs, _, err := h.logRepo.GetByJobID(id, 1, 10)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type ExecutionRow struct {
		ScheduleTime  string `json:"schedule_time"`
		StartTime     string `json:"start_time"`
		EndTime       string `json:"end_time"`
		ExecutionTime string `json:"execution_time"`
		HTTPCode      string `json:"http_code"`
		Status        string `json:"status"`
		Output        string `json:"output"`
		IsScheduled   bool   `json:"is_scheduled"`
	}

	var result []ExecutionRow

	for _, l := range logs {
		execSec := float64(l.ExecutionTimeMS) / 1000.0
		statusStr := "Failed"
		if l.StatusCode >= 200 && l.StatusCode < 300 {
			statusStr = "Succeeded"
		}
		endTime := l.ExecutedAt.Add(time.Duration(l.ExecutionTimeMS) * time.Millisecond)
		outputStr := l.ResponseBodySnippet
		if l.ErrorMessage != "" {
			outputStr = l.ErrorMessage
		}

		httpCodeStr := strconv.Itoa(l.StatusCode)
		if l.StatusCode == 0 {
			httpCodeStr = "ERR"
		}

		result = append(result, ExecutionRow{
			ScheduleTime:  l.ExecutedAt.Format("2006-01-02 15:04:05"),
			StartTime:     l.ExecutedAt.Format("15:04:05"),
			EndTime:       endTime.Format("15:04:05"),
			ExecutionTime: fmt.Sprintf("%.6f", execSec),
			HTTPCode:      httpCodeStr,
			Status:        statusStr,
			Output:        outputStr,
			IsScheduled:   false,
		})
	}

	// 2. Predict next 10 future executions
	sched, _, err := model.ParseCronExpression(job.CronExpression)
	if err == nil {
		curr := time.Now()
		for i := 0; i < 10; i++ {
			next := sched.Next(curr)
			if next.IsZero() {
				break
			}
			result = append(result, ExecutionRow{
				ScheduleTime:  next.Format("2006-01-02 15:04:05"),
				StartTime:     "-",
				EndTime:       "-",
				ExecutionTime: "-",
				HTTPCode:      "-",
				Status:        "Scheduled",
				Output:        "-",
				IsScheduled:   true,
			})
			curr = next
		}
	}

	h.respondJSON(w, r, http.StatusOK, map[string]interface{}{
		"cron_job_id":   job.ID,
		"cron_job_name": job.Name,
		"url":           job.URL,
		"executions":    result,
	})
}
