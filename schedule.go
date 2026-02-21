package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

// Job represents a scheduled task.
type Job struct {
	ID       int
	Interval time.Duration
	Task     func() error
	NextRun  time.Time
	RunOnce  bool
	Tags     []string
}

// persistentJob is used for JSON persistence.
type persistentJob struct {
	ID       int           `json:"id"`
	Interval time.Duration `json:"interval"`
	NextRun  time.Time     `json:"next_run"`
	RunOnce  bool          `json:"run_once"`
	Tags     []string      `json:"tags"`
}

// Scheduler manages multiple jobs.
type Scheduler struct {
	jobs      []*Job
	mu        sync.Mutex
	stop      chan struct{}
	jobID     int
	lastError error
}

// New creates a new Scheduler.
func New() *Scheduler {
	return &Scheduler{
		stop: make(chan struct{}),
	}
}

// Every defines the interval number.
func (s *Scheduler) Every(n int) *IntervalBuilder {
	return &IntervalBuilder{
		scheduler: s,
		interval:  n,
	}
}

// IntervalBuilder allows chaining time units.
type IntervalBuilder struct {
	scheduler *Scheduler
	interval  int
	duration  time.Duration
	runOnce   bool
	tags      []string
}

// Seconds sets interval in seconds.
func (b *IntervalBuilder) Seconds() *IntervalBuilder {
	b.duration = time.Duration(b.interval) * time.Second
	return b
}

// Minutes sets interval in minutes.
func (b *IntervalBuilder) Minutes() *IntervalBuilder {
	b.duration = time.Duration(b.interval) * time.Minute
	return b
}

// Hours sets interval in hours.
func (b *IntervalBuilder) Hours() *IntervalBuilder {
	b.duration = time.Duration(b.interval) * time.Hour
	return b
}

// Days sets interval in days.
func (b *IntervalBuilder) Days() *IntervalBuilder {
	b.duration = time.Duration(b.interval) * 24 * time.Hour
	return b
}

// Weeks sets interval in weeks.
func (b *IntervalBuilder) Weeks() *IntervalBuilder {
	b.duration = time.Duration(b.interval) * 7 * 24 * time.Hour
	return b
}

// RunOnce makes the job execute only once.
func (b *IntervalBuilder) RunOnce() *IntervalBuilder {
	b.runOnce = true
	return b
}

// Tag assigns tags to the job.
func (b *IntervalBuilder) Tag(tags ...string) *IntervalBuilder {
	b.tags = append(b.tags, tags...)
	return b
}

// Do registers the job.
func (b *IntervalBuilder) Do(task func() error) (*Job, error) {
	if b.duration <= 0 {
		return nil, errors.New("invalid duration: must specify time unit (Seconds, Minutes, etc.)")
	}

	if task == nil {
		return nil, errors.New("task cannot be nil")
	}

	s := b.scheduler

	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobID++

	job := &Job{
		ID:       s.jobID,
		Interval: b.duration,
		Task:     task,
		NextRun:  time.Now().Add(b.duration),
		RunOnce:  b.runOnce,
		Tags:     b.tags,
	}

	s.jobs = append(s.jobs, job)

	return job, nil
}

// Run starts the scheduler loop.
func (s *Scheduler) Run() {
	s.RunWithContext(context.Background())
}

// RunWithContext runs scheduler with cancellation support.
func (s *Scheduler) RunWithContext(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runPending()
		case <-s.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stop)
}

// LastError returns the last job error encountered.
func (s *Scheduler) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastError
}

// RemoveByTag removes jobs matching a tag.
func (s *Scheduler) RemoveByTag(tag string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var filtered []*Job
	for _, job := range s.jobs {
		if !contains(job.Tags, tag) {
			filtered = append(filtered, job)
		}
	}
	s.jobs = filtered
}

// runPending executes due jobs.
func (s *Scheduler) runPending() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var remaining []*Job

	for _, job := range s.jobs {
		if now.After(job.NextRun) || now.Equal(job.NextRun) {
			go func(j *Job) {
				if err := j.Task(); err != nil {
					s.mu.Lock()
					s.lastError = err
					s.mu.Unlock()
				}
			}(job)

			if job.RunOnce {
				continue
			}

			job.NextRun = now.Add(job.Interval)
		}
		remaining = append(remaining, job)
	}

	s.jobs = remaining
}

// SaveToFile persists scheduler jobs to JSON file (without task functions).
func (s *Scheduler) SaveToFile(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var pj []persistentJob
	for _, job := range s.jobs {
		pj = append(pj, persistentJob{
			ID:       job.ID,
			Interval: job.Interval,
			NextRun:  job.NextRun,
			RunOnce:  job.RunOnce,
			Tags:     job.Tags,
		})
	}

	data, err := json.MarshalIndent(pj, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile restores job metadata (requires re-attaching task functions manually).
func (s *Scheduler) LoadFromFile(filename string, taskResolver func(id int) func() error) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var pj []persistentJob
	if err := json.Unmarshal(data, &pj); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range pj {
		task := taskResolver(p.ID)
		if task == nil {
			continue
		}

		job := &Job{
			ID:       p.ID,
			Interval: p.Interval,
			NextRun:  p.NextRun,
			RunOnce:  p.RunOnce,
			Tags:     p.Tags,
			Task:     task,
		}

		s.jobs = append(s.jobs, job)

		if p.ID > s.jobID {
			s.jobID = p.ID
		}
	}

	return nil
}

// contains checks if slice contains string.
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
