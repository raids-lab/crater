package reconciler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	profileQueueCapacity = 256
	profileWorkerCount   = 4
	profileRecentWindow  = 2 * time.Minute
)

type vcJobProfileTask struct {
	namespace, jobName         string
	errorRetries, emptyRetries int
}

type vcJobProfileQueue struct {
	jobs                       chan vcJobProfileTask
	mu                         sync.Mutex
	pending                    map[string]struct{}
	stopped                    bool
	clock                      clock.Clock
	errorBackoff, emptyBackoff []time.Duration
}

func newVCJobProfileQueue() *vcJobProfileQueue {
	return &vcJobProfileQueue{
		jobs: make(chan vcJobProfileTask, profileQueueCapacity), pending: make(map[string]struct{}),
		clock: clock.RealClock{}, errorBackoff: []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute},
		emptyBackoff: []time.Duration{30 * time.Second, time.Minute},
	}
}

func (q *vcJobProfileQueue) enqueue(task vcJobProfileTask) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, exists := q.pending[task.jobName]; exists {
		return true
	}
	// 等待重试的任务也占用容量，避免故障时累积无界计时器。
	if q.stopped || len(q.pending) >= cap(q.jobs) {
		return false
	}
	// Every queued, active or retrying task owns a pending entry. Here
	// len(jobs) <= len(pending) < cap(jobs), so this send cannot block.
	q.pending[task.jobName] = struct{}{}
	q.jobs <- task
	return true
}

func (q *vcJobProfileQueue) finish(task vcJobProfileTask) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.pending, task.jobName)
}

func (q *vcJobProfileQueue) run(ctx context.Context, collect func(context.Context, *vcJobProfileTask) time.Duration) error {
	var workers, retries sync.WaitGroup
	for range profileWorkerCount {
		workers.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case task := <-q.jobs:
					if ctx.Err() != nil {
						return
					}
					delay := q.process(ctx, &task, collect)
					if delay == 0 || ctx.Err() != nil {
						q.finish(task)
						continue
					}
					// 重试等待释放 worker；任务始终保留在去重集合中。
					retries.Go(func() {
						timer := q.clock.NewTimer(delay)
						defer timer.Stop()
						select {
						case <-ctx.Done():
						case <-timer.C():
							// This retry owns a pending entry and is absent from jobs,
							// leaving a channel slot even when pending is at capacity.
							select {
							case q.jobs <- task:
							case <-ctx.Done():
							}
						}
					})
				}
			}
		})
	}
	<-ctx.Done()
	q.mu.Lock()
	q.stopped = true
	q.mu.Unlock()
	workers.Wait()
	retries.Wait()
	q.mu.Lock()
	clear(q.pending)
	q.mu.Unlock()
	return nil
}

func (*vcJobProfileQueue) process(
	ctx context.Context, task *vcJobProfileTask, collect func(context.Context, *vcJobProfileTask) time.Duration,
) (delay time.Duration) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.FromContext(ctx).Error(fmt.Errorf("profile worker panic: %v", recovered),
				"job profile task panicked", "job", task.jobName)
		}
	}()
	return collect(ctx, task)
}
