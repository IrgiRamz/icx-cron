package repository

import (
	"database/sql"
	"fmt"

	"iconix-cron/internal/model"
)

type LogRepository struct {
	db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) Create(log *model.JobLog) error {
	query := `
	INSERT INTO job_logs (job_id, status_code, execution_time_ms, error_message, response_body_snippet, executed_at)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	res, err := r.db.Exec(query, log.JobID, log.StatusCode, log.ExecutionTimeMS, log.ErrorMessage, log.ResponseBodySnippet)
	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	log.ID = id
	return nil
}

func (r *LogRepository) GetByJobID(jobID int64, page, pageSize int) ([]model.JobLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM job_logs WHERE job_id = ?", jobID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
	SELECT id, job_id, status_code, execution_time_ms, error_message, response_body_snippet, executed_at
	FROM job_logs
	WHERE job_id = ?
	ORDER BY executed_at DESC
	LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, jobID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.JobLog
	for rows.Next() {
		var l model.JobLog
		if err := rows.Scan(&l.ID, &l.JobID, &l.StatusCode, &l.ExecutionTimeMS, &l.ErrorMessage, &l.ResponseBodySnippet, &l.ExecutedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}

func (r *LogRepository) GetAllLogs(page, pageSize int) ([]model.JobLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 25
	}
	offset := (page - 1) * pageSize

	var total int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM job_logs").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
	SELECT l.id, l.job_id, COALESCE(j.name, 'Deleted Job') as job_name, COALESCE(j.url, '') as job_url,
	       l.status_code, l.execution_time_ms, l.error_message, l.response_body_snippet, l.executed_at
	FROM job_logs l
	LEFT JOIN jobs j ON l.job_id = j.id
	ORDER BY l.executed_at DESC
	LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.JobLog
	for rows.Next() {
		var l model.JobLog
		if err := rows.Scan(&l.ID, &l.JobID, &l.JobName, &l.JobURL, &l.StatusCode, &l.ExecutionTimeMS, &l.ErrorMessage, &l.ResponseBodySnippet, &l.ExecutedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}

// PruneOlderThan deletes execution logs older than the specified retention duration (e.g. 7 days)
func (r *LogRepository) PruneOlderThan(retentionDays int) (int64, error) {
	query := `DELETE FROM job_logs WHERE executed_at < DATETIME('now', fmt_days)`
	fmtDays := fmt.Sprintf("-%d days", retentionDays)
	query = fmt.Sprintf(`DELETE FROM job_logs WHERE executed_at < DATETIME('now', '%s')`, fmtDays)

	res, err := r.db.Exec(query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
