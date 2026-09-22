package scheduler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"iconix-cron/internal/model"
	"iconix-cron/internal/repository"
)

type Executor struct {
	client     *http.Client
	jobRepo    *repository.JobRepository
	logRepo    *repository.LogRepository
	semaphore  chan struct{}
}

func NewExecutor(jobRepo *repository.JobRepository, logRepo *repository.LogRepository, maxWorkers int) *Executor {
	if maxWorkers <= 0 {
		maxWorkers = 30
	}

	// Optimized HTTP Transport for 150+ endpoints with Keep-Alive connection pooling
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
	}

	return &Executor{
		client:    client,
		jobRepo:   jobRepo,
		logRepo:   logRepo,
		semaphore: make(chan struct{}, maxWorkers),
	}
}

func (e *Executor) ExecuteJob(job *model.Job) {
	// Acquire worker slot from semaphore
	e.semaphore <- struct{}{}
	go func() {
		defer func() {
			<-e.semaphore // Release worker slot
		}()

		e.runSingleJob(job)
	}()
}

func (e *Executor) runSingleJob(job *model.Job) {
	start := time.Now()

	timeout := time.Duration(job.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	method := job.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, job.URL, nil)
	if err != nil {
		e.recordResult(job, 0, time.Since(start).Milliseconds(), fmt.Sprintf("Failed to create request: %v", err), "")
		return
	}

	req.Header.Set("User-Agent", "IconixCron-Scheduler/1.0 (+https://iconix.co.id)")

	resp, err := e.client.Do(req)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		e.recordResult(job, 0, durationMs, err.Error(), "")
		return
	}

	// Read first 512 bytes for snippet and discard remaining to allow Keep-Alive connection reuse
	snippetBuf := make([]byte, 512)
	n, _ := io.ReadFull(resp.Body, snippetBuf)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close() // Mandatory closure to release socket back to transport pool

	snippet := string(snippetBuf[:n])
	isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 300

	errMsg := ""
	if !isSuccess {
		errMsg = fmt.Sprintf("HTTP Status Code %d", resp.StatusCode)
	}

	e.recordResult(job, resp.StatusCode, durationMs, errMsg, snippet)
}

func (e *Executor) recordResult(job *model.Job, statusCode int, durationMs int64, errMsg string, snippet string) {
	isSuccess := statusCode >= 200 && statusCode < 300

	// Save log record
	logItem := &model.JobLog{
		JobID:               job.ID,
		StatusCode:          statusCode,
		ExecutionTimeMS:     durationMs,
		ErrorMessage:        errMsg,
		ResponseBodySnippet: snippet,
		ExecutedAt:          time.Now(),
	}

	if err := e.logRepo.Create(logItem); err != nil {
		log.Printf("[EXECUTOR] Error saving log for job %d: %v", job.ID, err)
	}

	// Update job execution stats in DB
	if err := e.jobRepo.UpdateExecutionStats(job.ID, isSuccess, time.Now(), nil); err != nil {
		log.Printf("[EXECUTOR] Error updating stats for job %d: %v", job.ID, err)
	}
}
