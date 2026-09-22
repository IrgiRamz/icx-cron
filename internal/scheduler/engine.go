package scheduler

import (
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
	"iconix-cron/internal/model"
	"iconix-cron/internal/repository"
)

type SchedulerEngine struct {
	cron       *cron.Cron
	executor   *Executor
	jobRepo    *repository.JobRepository
	logRepo    *repository.LogRepository
	entryMap   map[int64]cron.EntryID
	mu         sync.RWMutex
	retention  int
}

func NewSchedulerEngine(jobRepo *repository.JobRepository, logRepo *repository.LogRepository, executor *Executor, retentionDays int) *SchedulerEngine {
	// Support optional seconds field for full compatibility (e.g. Easycron 6-field specs)
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	c := cron.New(cron.WithParser(parser))

	return &SchedulerEngine{
		cron:      c,
		executor:  executor,
		jobRepo:   jobRepo,
		logRepo:   logRepo,
		entryMap:  make(map[int64]cron.EntryID),
		retention: retentionDays,
	}
}

func (s *SchedulerEngine) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load active jobs from DB
	activeJobs, err := s.jobRepo.GetActiveJobs()
	if err != nil {
		return fmt.Errorf("failed to fetch active jobs: %w", err)
	}

	for i := range activeJobs {
		job := activeJobs[i]
		if err := s.addJobToCron(&job); err != nil {
			log.Printf("[SCHEDULER] Warning: failed to schedule job ID %d (%s): %v", job.ID, job.Name, err)
		}
	}

	// Schedule 7-day Log Auto-Pruning job daily at 02:00 AM
	_, err = s.cron.AddFunc("0 0 2 * * *", func() {
		pruned, err := s.logRepo.PruneOlderThan(s.retention)
		if err != nil {
			log.Printf("[SCHEDULER] Error during log pruning: %v", err)
		} else {
			log.Printf("[SCHEDULER] Auto-pruned %d log rows older than %d days.", pruned, s.retention)
		}
	})
	if err != nil {
		log.Printf("[SCHEDULER] Warning: failed to schedule log pruner: %v", err)
	}

	s.cron.Start()
	log.Printf("[SCHEDULER] Scheduler Engine started with %d active jobs.", len(s.entryMap))
	return nil
}

func (s *SchedulerEngine) Stop() {
	s.cron.Stop()
	log.Println("[SCHEDULER] Scheduler Engine stopped.")
}

func (s *SchedulerEngine) addJobToCron(job *model.Job) error {
	if job.Status != 1 {
		return nil
	}

	sched, _, err := model.ParseCronExpression(job.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", job.CronExpression, err)
	}

	// Capture job pointer safely in closure
	jobCopy := *job
	entryID := s.cron.Schedule(sched, cron.FuncJob(func() {
		// Re-fetch latest job config from DB if needed, or pass current snapshot
		latestJob, err := s.jobRepo.GetByID(jobCopy.ID)
		if err != nil || latestJob == nil || latestJob.Status != 1 {
			return
		}
		s.executor.ExecuteJob(latestJob)
	}))

	s.entryMap[job.ID] = entryID
	return nil
}

func (s *SchedulerEngine) ReloadJob(jobID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if present
	if entryID, exists := s.entryMap[jobID]; exists {
		s.cron.Remove(entryID)
		delete(s.entryMap, jobID)
	}

	// Re-fetch job from DB
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		return err
	}
	if job == nil || job.Status != 1 {
		return nil // Paused or deleted
	}

	return s.addJobToCron(job)
}

func (s *SchedulerEngine) RemoveJob(jobID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.entryMap[jobID]; exists {
		s.cron.Remove(entryID)
		delete(s.entryMap, jobID)
	}
}

func (s *SchedulerEngine) TriggerJobNow(jobID int64) error {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	s.executor.ExecuteJob(job)
	return nil
}
