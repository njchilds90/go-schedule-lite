package schedule

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunsJob(t *testing.T) {
	s := New()

	var counter int32

	s.Every(1).Seconds().Do(func() {
		atomic.AddInt32(&counter, 1)
	})

	go s.Run()
	time.Sleep(2500 * time.Millisecond)
	s.Stop()

	if atomic.LoadInt32(&counter) == 0 {
		t.Errorf("Expected job to run at least once")
	}
}
