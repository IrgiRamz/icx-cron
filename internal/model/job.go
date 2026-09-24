package model

import "time"

// Job represents a Cron Job configuration
type Job struct {
	ID                 int64      `json:"cron_job_id" db:"id"`
	Name               string     `json:"cron_job_name" db:"name"`
	URL                string     `json:"url" db:"url"`
	Method             string     `json:"http_method" db:"method"`
	CronExpression     string     `json:"cron_expression" db:"cron_expression"`
	Status             int        `json:"status" db:"status"` // 1: active, 0: paused
	TimeoutSeconds     int        `json:"timeout" db:"timeout_seconds"`
	RetryCount         int        `json:"retry_count" db:"retry_count"`
	HTTPHeaders        string     `json:"http_headers" db:"http_headers"`
	HTTPMessageBody    string     `json:"http_message_body" db:"http_message_body"`
	AuthUser           string     `json:"http_auth_user" db:"auth_user"`
	AuthPW             string     `json:"http_auth_pw" db:"auth_pw"`
	TotalSuccess       int64      `json:"total_successes" db:"total_success"`
	TotalFail          int64      `json:"total_failures" db:"total_fail"`
	ConsecutiveFail    int64      `json:"current_failures" db:"consecutive_fail"`
	LastRunAt          *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt          *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// JobLog represents an execution log for a job
type JobLog struct {
	ID                  int64     `json:"log_id" db:"id"`
	JobID               int64     `json:"cron_job_id" db:"job_id"`
	JobName             string    `json:"cron_job_name,omitempty" db:"job_name"`
	JobURL              string    `json:"url,omitempty" db:"job_url"`
	StatusCode          int       `json:"http_status_code" db:"status_code"`
	ExecutionTimeMS     int64     `json:"execution_time_ms" db:"execution_time_ms"`
	ErrorMessage        string    `json:"error_message,omitempty" db:"error_message"`
	ResponseBodySnippet string    `json:"response_snippet,omitempty" db:"response_body_snippet"`
	ExecutedAt          time.Time `json:"executed_at" db:"executed_at"`
}

// LogItemResponse represents an Easycron compatible response serializer for execution logs
type LogItemResponse struct {
	LogID         int64   `json:"log_id,omitempty"`
	CronJobID     int64   `json:"cron_job_id"`
	ScheduledTime string  `json:"scheduled_time"`
	FiredTime     string  `json:"fired_time"`
	DoneTime      string  `json:"done_time"`
	TotalTime     float64 `json:"total_time"` // Konversi dari execution_time_ms ke detik
	HTTPCode      int     `json:"http_code"`  // Dari http_status_code
	Status        string  `json:"status"`     // "Succeeded" jika 2xx/3xx, "Failed" jika >=400 atau error
	Error         string  `json:"error"`
	BotIP         string  `json:"bot_ip,omitempty"`

	// Opsional: Pertahankan field lama sebagai fallback agar backward compatible
	HTTPStatusCode  int    `json:"http_status_code,omitempty"`
	ExecutionTimeMs int64  `json:"execution_time_ms,omitempty"`
	ExecutedAt      string `json:"executed_at,omitempty"`
}

// ToLogItemResponse converts a JobLog database model to LogItemResponse
func (l *JobLog) ToLogItemResponse() LogItemResponse {
	statusStr := "Failed"
	if l.StatusCode >= 200 && l.StatusCode < 400 && l.ErrorMessage == "" {
		statusStr = "Succeeded"
	}

	doneTime := l.ExecutedAt.Add(time.Duration(l.ExecutionTimeMS) * time.Millisecond)

	return LogItemResponse{
		LogID:           l.ID,
		CronJobID:       l.JobID,
		ScheduledTime:   l.ExecutedAt.Format("2006-01-02 15:04:05"),
		FiredTime:       l.ExecutedAt.Format("2006-01-02 15:04:05"),
		DoneTime:        doneTime.Format("2006-01-02 15:04:05"),
		TotalTime:       float64(l.ExecutionTimeMS) / 1000.0,
		HTTPCode:        l.StatusCode,
		Status:          statusStr,
		Error:           l.ErrorMessage,
		HTTPStatusCode:  l.StatusCode,
		ExecutionTimeMs: l.ExecutionTimeMS,
		ExecutedAt:      l.ExecutedAt.Format("2006-01-02 15:04:05"),
	}
}

// EasycronJobResponse represents the Easycron API compatible JSON structure for a job
type EasycronJobResponse struct {
	CronJobID         int64    `json:"cron_job_id"`
	URL               string   `json:"url"`
	HTTPAuthUser      string   `json:"http_auth_user"`
	HTTPAuthPW        string   `json:"http_auth_pw"`
	CronExpression    string   `json:"cron_expression"`
	Timezone          string   `json:"timezone"`
	HTTPMethod        string   `json:"http_method"`
	HTTPHeaders       string   `json:"http_headers"`
	HTTPMessageBody   string   `json:"http_message_body"`
	Timeout           int      `json:"timeout"`
	SuccessCriterion  int      `json:"success_criterion"`
	SuccessRegexp     string   `json:"success_regexp"`
	FailureRegexp     string   `json:"failure_regexp"`
	SendEmail         int      `json:"send_email"`
	EmailThreshold    int      `json:"email_threshold"`
	SendSlack         int      `json:"send_slack"`
	SlackThreshold    int      `json:"slack_threshold"`
	SlackURL          string   `json:"slack_url"`
	SendWebhook       int      `json:"send_webhook"`
	WebhookHTTPMethod string   `json:"webhook_http_method"`
	WebhookURL        string   `json:"webhook_url"`
	WebhookData       []string `json:"webhook_data"`
	Status            int      `json:"status"`
	EPDSOccupied      int      `json:"epds_occupied"`
	CronJobName       string   `json:"cron_job_name"`
	Description       string   `json:"description"`
	GroupID           int64    `json:"group_id"`
	TotalSuccesses    int64    `json:"total_successes"`
	TotalFailures     int64    `json:"total_failures"`
	CurrentFailures   int64    `json:"current_failures"`
}

// ToEasycronResponse maps Job model to EasycronJobResponse format
func (j *Job) ToEasycronResponse() EasycronJobResponse {
	return EasycronJobResponse{
		CronJobID:        j.ID,
		URL:              j.URL,
		HTTPAuthUser:     j.AuthUser,
		HTTPAuthPW:       j.AuthPW,
		CronExpression:   j.CronExpression,
		Timezone:         "Asia/Jakarta",
		HTTPMethod:       j.Method,
		HTTPHeaders:      j.HTTPHeaders,
		HTTPMessageBody:  j.HTTPMessageBody,
		Timeout:          j.TimeoutSeconds,
		SuccessCriterion: 1,
		SuccessRegexp:    "",
		FailureRegexp:    "",
		SendEmail:        0,
		EmailThreshold:   1,
		SendSlack:        0,
		SlackThreshold:   1,
		SlackURL:         "",
		SendWebhook:      0,
		WebhookHTTPMethod: "GET",
		WebhookURL:       "",
		WebhookData:      []string{},
		Status:           j.Status,
		EPDSOccupied:     int(j.GetEPD()),
		CronJobName:      j.Name,
		Description:      "",
		GroupID:          0,
		TotalSuccesses:   j.TotalSuccess,
		TotalFailures:    j.TotalFail,
		CurrentFailures:  j.ConsecutiveFail,
	}
}
