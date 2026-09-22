package api

import (
	"html/template"
	"net/http"

	"iconix-cron/internal/repository"
	"iconix-cron/web"
)

type WebHandler struct {
	jobRepo *repository.JobRepository
	logRepo *repository.LogRepository
	tmpl    *template.Template
}

func NewWebHandler(jobRepo *repository.JobRepository, logRepo *repository.LogRepository) (*WebHandler, error) {
	tmpl, err := template.ParseFS(web.TemplateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &WebHandler{
		jobRepo: jobRepo,
		logRepo: logRepo,
		tmpl:    tmpl,
	}, nil
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
