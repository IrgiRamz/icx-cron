package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"iconix-cron/internal/model"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(job *model.Job) error {
	query := `
	INSERT INTO jobs (name, url, method, cron_expression, status, timeout_seconds, retry_count, http_headers, http_message_body, auth_user, auth_pw, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	if strings.TrimSpace(job.Name) == "" {
		job.Name = "Unnamed"
	}
	if job.Method == "" {
		job.Method = "GET"
	}
	if job.TimeoutSeconds <= 0 {
		job.TimeoutSeconds = 10
	}

	res, err := r.db.Exec(query, job.Name, job.URL, job.Method, job.CronExpression, job.Status, job.TimeoutSeconds, job.RetryCount, job.HTTPHeaders, job.HTTPMessageBody, job.AuthUser, job.AuthPW)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	job.ID = id
	return nil
}

func (r *JobRepository) GetByID(id int64) (*model.Job, error) {
	query := `
	SELECT id, name, url, method, cron_expression, status, timeout_seconds, retry_count,
	       total_success, total_fail, consecutive_fail, last_run_at, next_run_at,
	       http_headers, http_message_body, auth_user, auth_pw, created_at, updated_at
	FROM jobs WHERE id = ?
	`
	row := r.db.QueryRow(query, id)
	var j model.Job
	var lastRun, nextRun sql.NullTime

	err := row.Scan(
		&j.ID, &j.Name, &j.URL, &j.Method, &j.CronExpression, &j.Status, &j.TimeoutSeconds, &j.RetryCount,
		&j.TotalSuccess, &j.TotalFail, &j.ConsecutiveFail, &lastRun, &nextRun,
		&j.HTTPHeaders, &j.HTTPMessageBody, &j.AuthUser, &j.AuthPW, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if lastRun.Valid {
		j.LastRunAt = &lastRun.Time
	}
	if nextRun.Valid {
		j.NextRunAt = &nextRun.Time
	}

	return &j, nil
}

func (r *JobRepository) GetAll(page, pageSize int, sortBy, order string) ([]model.Job, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var total int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	sortColumn := "id"
	switch sortBy {
	case "cron_job_id", "id":
		sortColumn = "id"
	case "cron_job_name", "name":
		sortColumn = "name"
	case "url":
		sortColumn = "url"
	case "total_successes", "total_success":
		sortColumn = "total_success"
	case "total_failures", "total_fail":
		sortColumn = "total_fail"
	}

	sortOrder := "DESC"
	if order == "asc" || order == "ASC" {
		sortOrder = "ASC"
	}

	query := fmt.Sprintf(`
	SELECT id, name, url, method, cron_expression, status, timeout_seconds, retry_count,
	       total_success, total_fail, consecutive_fail, last_run_at, next_run_at,
	       http_headers, http_message_body, auth_user, auth_pw, created_at, updated_at
	FROM jobs
	ORDER BY %s %s
	LIMIT ? OFFSET ?
	`, sortColumn, sortOrder)

	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []model.Job
	for rows.Next() {
		var j model.Job
		var lastRun, nextRun sql.NullTime
		if err := rows.Scan(
			&j.ID, &j.Name, &j.URL, &j.Method, &j.CronExpression, &j.Status, &j.TimeoutSeconds, &j.RetryCount,
			&j.TotalSuccess, &j.TotalFail, &j.ConsecutiveFail, &lastRun, &nextRun,
			&j.HTTPHeaders, &j.HTTPMessageBody, &j.AuthUser, &j.AuthPW, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if lastRun.Valid {
			j.LastRunAt = &lastRun.Time
		}
		if nextRun.Valid {
			j.NextRunAt = &nextRun.Time
		}
		jobs = append(jobs, j)
	}

	return jobs, total, nil
}

func (r *JobRepository) GetActiveJobs() ([]model.Job, error) {
	query := `
	SELECT id, name, url, method, cron_expression, status, timeout_seconds, retry_count,
	       total_success, total_fail, consecutive_fail, last_run_at, next_run_at,
	       http_headers, http_message_body, auth_user, auth_pw, created_at, updated_at
	FROM jobs WHERE status = 1
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.Job
	for rows.Next() {
		var j model.Job
		var lastRun, nextRun sql.NullTime
		if err := rows.Scan(
			&j.ID, &j.Name, &j.URL, &j.Method, &j.CronExpression, &j.Status, &j.TimeoutSeconds, &j.RetryCount,
			&j.TotalSuccess, &j.TotalFail, &j.ConsecutiveFail, &lastRun, &nextRun,
			&j.HTTPHeaders, &j.HTTPMessageBody, &j.AuthUser, &j.AuthPW, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if lastRun.Valid {
			j.LastRunAt = &lastRun.Time
		}
		if nextRun.Valid {
			j.NextRunAt = &nextRun.Time
		}
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (r *JobRepository) Update(job *model.Job) error {
	query := `
	UPDATE jobs SET name = ?, url = ?, method = ?, cron_expression = ?, status = ?,
	               timeout_seconds = ?, retry_count = ?, http_headers = ?, http_message_body = ?,
	               auth_user = ?, auth_pw = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`
	_, err := r.db.Exec(query, job.Name, job.URL, job.Method, job.CronExpression, job.Status, job.TimeoutSeconds, job.RetryCount, job.HTTPHeaders, job.HTTPMessageBody, job.AuthUser, job.AuthPW, job.ID)
	return err
}

func (r *JobRepository) UpdateStatus(id int64, status int) error {
	query := `UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *JobRepository) UpdateExecutionStats(id int64, success bool, lastRun time.Time, nextRun *time.Time) error {
	var query string
	if success {
		query = `
		UPDATE jobs
		SET total_success = total_success + 1,
		    consecutive_fail = 0,
		    last_run_at = ?,
		    next_run_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		`
	} else {
		query = `
		UPDATE jobs
		SET total_fail = total_fail + 1,
		    consecutive_fail = consecutive_fail + 1,
		    last_run_at = ?,
		    next_run_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		`
	}

	var nextRunNull sql.NullTime
	if nextRun != nil {
		nextRunNull = sql.NullTime{Time: *nextRun, Valid: true}
	}

	_, err := r.db.Exec(query, lastRun, nextRunNull, id)
	return err
}

func (r *JobRepository) Delete(id int64) error {
	query := `DELETE FROM jobs WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *JobRepository) ResetStats(id int64) error {
	query := `UPDATE jobs SET total_success = 0, total_fail = 0, consecutive_fail = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}
