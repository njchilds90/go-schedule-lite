package schedule

import (
	"sync"
	"time"
)

// Job represents a scheduled task.
type Job struct {
	interval time.Duration
	task     func()
	nextRun  time.Time
}

// Scheduler manages multiple jobs.
type Scheduler struct {
	jobs []*Job
	mu   sync.Mutex
	stop chan struct{}
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

// Do registers the job.
func (b *IntervalBuilder) Do(task func()) {
	job := &Job{
		interval: b.duration,
		task:     task,
		nextRun:  time.Now().Add(b.duration),
	}

	b.scheduler.mu.Lock()
	defer b.scheduler.mu.Unlock()
	b.scheduler.jobs = append(b.scheduler.jobs, job)
}

// Run starts the scheduler loop.
func (s *Scheduler) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runPending()
		case <-s.stop:
			return
		}
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stop)
}

// runPending executes due jobs.
func (s *Scheduler) runPending() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, job := range s.jobs {
		if now.After(job.nextRun) {
			go job.task()
			job.nextRun = now.Add(job.interval)
		}
	}
}
