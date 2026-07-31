package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	// systemTaskRunnerIdleInterval is the fallback poll interval used to pick up
	// tasks created on other nodes and mark expired leases failed.
	systemTaskRunnerIdleInterval = 15 * time.Second
	systemTaskLockTTL            = 60 * time.Second

	// systemTaskSchedulerInterval throttles how often the scheduler/stale-lock
	// pass runs, independent of how often the runner wakes to claim tasks.
	systemTaskSchedulerInterval = 15 * time.Second
	systemTaskStaleLockInterval = 30 * time.Second
)

// SystemTaskHandler executes a claimed task of a specific type. Run owns the
// task lifecycle from claim to terminal state: it MUST call
// model.FinishSystemTask (succeeded/failed) before returning and MUST honor
// ctx cancellation, which the runner triggers if the per-type lock is lost.
type SystemTaskHandler interface {
	Type() string
	Run(ctx context.Context, task *model.SystemTask, runnerID string)
}

// ScheduledSystemTaskHandler is a SystemTaskHandler that the scheduler also
// creates periodically when enabled and the configured interval has elapsed
// since the last run.
type ScheduledSystemTaskHandler interface {
	SystemTaskHandler
	Enabled() bool
	Interval() time.Duration
	NewPayload() any
}

var (
	systemTaskHandlersMu sync.RWMutex
	systemTaskHandlers   = map[string]SystemTaskHandler{}
)

// RegisterSystemTaskHandler registers a handler keyed by its Type(). It must be
// called before StartSystemTaskRunner (or any time, since the runner snapshots
// the registry every pass). Re-registering a type replaces the previous handler.
func RegisterSystemTaskHandler(h SystemTaskHandler) {
	if h == nil {
		return
	}
	systemTaskHandlersMu.Lock()
	defer systemTaskHandlersMu.Unlock()
	systemTaskHandlers[h.Type()] = h
}

func registeredSystemTaskHandlers() []SystemTaskHandler {
	systemTaskHandlersMu.RLock()
	defer systemTaskHandlersMu.RUnlock()
	handlers := make([]SystemTaskHandler, 0, len(systemTaskHandlers))
	for _, h := range systemTaskHandlers {
		handlers = append(handlers, h)
	}
	return handlers
}

// systemTaskWakeup signals the runner to check for runnable tasks immediately
// instead of waiting for the idle poll. It is buffered so a signal raised while
// the runner is busy is handled on the next loop.
var systemTaskWakeup = make(chan struct{}, 1)

// WakeSystemTaskRunner wakes the runner without blocking. If a wakeup is
// already pending it is a no-op, which is fine since one pass drains all work.
func WakeSystemTaskRunner() {
	select {
	case systemTaskWakeup <- struct{}{}:
	default:
	}
}

func StartSystemTaskRunner() {
	startSystemTaskRunner()
}

// EnqueueSystemTask creates an on-demand task of the given type. The returned
// bool is true only when a new pending row was created; false means an active
// task of the same type already exists and was returned.
func EnqueueSystemTask(taskType string, payload any) (*model.SystemTask, bool, error) {
	activeTask, err := model.GetActiveSystemTask(taskType)
	if err != nil {
		return nil, false, err
	}
	if activeTask != nil {
		return activeTask, false, nil
	}

	task, err := model.CreateSystemTask(taskType, payload, nil)
	if err != nil {
		activeTask, activeErr := model.GetActiveSystemTask(taskType)
		if activeErr == nil && activeTask != nil {
			return activeTask, false, nil
		}
		return nil, false, err
	}
	WakeSystemTaskRunner()
	return task, true, nil
}

// runSystemTaskClaimPass tries to claim one pending task per registered type
// and dispatches each claimed task in its own goroutine so a long-running
// handler (e.g. channel test) never blocks another type (e.g. log cleanup).
func runSystemTaskClaimPass(ctx context.Context, runnerID string) {
	if ctx.Err() != nil {
		return
	}
	handlers := registeredSystemTaskHandlers()
	taskTypes := make([]string, 0, len(handlers))
	for _, handler := range handlers {
		if ctx.Err() != nil {
			return
		}
		taskTypes = append(taskTypes, handler.Type())
	}
	pendingTasks, err := model.FindEarliestPendingSystemTasks(taskTypes)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("system task runner query failed: %v", err))
		return
	}
	for _, handler := range handlers {
		task := pendingTasks[handler.Type()]
		if task == nil {
			continue
		}
		claimedTask, claimed, err := model.ClaimSystemTask(task.ID, handler.Type(), runnerID, systemTaskLockUntil())
		if err != nil {
			logger.LogWarn(context.Background(), fmt.Sprintf("system task claim failed: %v", err))
			continue
		}
		if !claimed {
			continue
		}
		dispatchHandler := handler
		dispatchTask := claimedTask
		systemTaskRunnerWG.Add(1)
		gopool.Go(func() {
			defer systemTaskRunnerWG.Done()
			runWithLeaseHeartbeat(ctx, dispatchTask, runnerID, func(ctx context.Context) {
				dispatchHandler.Run(ctx, dispatchTask, runnerID)
			})
		})
	}
}

// runSystemTaskScheduler creates a new task row for each enabled scheduled
// handler whose interval has elapsed since its last run and that has no active
// row. The task active_key unique index deduplicates concurrent creation while
// the per-type lock guarantees only one runner executes the task.
func runSystemTaskScheduler() {
	now := common.GetTimestamp()
	handlers := registeredSystemTaskHandlers()
	scheduledHandlers := make([]ScheduledSystemTaskHandler, 0, len(handlers))
	taskTypes := make([]string, 0, len(handlers))
	for _, handler := range handlers {
		scheduled, ok := handler.(ScheduledSystemTaskHandler)
		if !ok || !scheduled.Enabled() {
			continue
		}
		scheduledHandlers = append(scheduledHandlers, scheduled)
		taskTypes = append(taskTypes, scheduled.Type())
	}
	latestTasks, err := model.GetLatestSystemTasks(taskTypes)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("system task scheduler query failed: %v", err))
		return
	}
	for _, scheduled := range scheduledHandlers {
		latest := latestTasks[scheduled.Type()]
		if latest != nil {
			if latest.Status == model.SystemTaskStatusPending || latest.Status == model.SystemTaskStatusRunning {
				continue // an active row already exists
			}
			if now-latest.UpdatedAt < int64(scheduled.Interval().Seconds()) {
				continue // not due yet
			}
		}
		if _, err := model.CreateSystemTask(scheduled.Type(), scheduled.NewPayload(), nil); err != nil {
			activeTask, activeErr := model.GetActiveSystemTask(scheduled.Type())
			if activeErr == nil && activeTask != nil {
				continue
			}
			if activeErr != nil {
				logger.LogWarn(context.Background(), fmt.Sprintf("system task scheduler active lookup failed: type=%s err=%v", scheduled.Type(), activeErr))
			}
			logger.LogWarn(context.Background(), fmt.Sprintf("system task scheduler create failed: type=%s err=%v", scheduled.Type(), err))
			continue
		}
	}
}

// runWithLeaseHeartbeat renews the per-type lock on a background ticker while
// fn runs. The TTL is a crash-detection window, not a task time limit: an
// arbitrarily long handler stays alive as long as the heartbeat succeeds.
func runWithLeaseHeartbeat(parent context.Context, task *model.SystemTask, runnerID string, fn func(ctx context.Context)) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	interval := systemTaskLockTTL / 3
	if interval <= 0 {
		interval = systemTaskLockTTL
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	done := make(chan struct{})
	heartbeatDone := make(chan struct{})

	go func() {
		defer close(heartbeatDone)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := model.RenewSystemTaskLock(task.TaskID, runnerID, systemTaskLockUntil()); err != nil {
					cancel()
					return
				}
			}
		}
	}()

	fn(ctx)
	close(done)
	<-heartbeatDone
}

func systemTaskLockUntil() int64 {
	return common.GetTimestamp() + int64(systemTaskLockTTL.Seconds())
}

// SystemTaskProgress is the state shape used by handlers that report percentage
// progress (channel test, model update). The frontend reads the progress field
// (0-100) to render a per-task progress indicator.
type SystemTaskProgress struct {
	Total     int `json:"total"`
	Processed int `json:"processed"`
	Progress  int `json:"progress"`
}

// NewSystemTaskProgressReporter returns a throttled progress callback bound to a
// running task. Handlers call it with (processed, total) as they iterate work;
// it persists a {processed,total,progress} state at most once every ~2s, always
// emitting the first update and the final 100%.
// Lock-loss errors are ignored: the lease heartbeat cancels the handler ctx on
// loss, so progress writes are best-effort and never abort the run themselves.
// The returned func is single-goroutine only (call it from the handler loop).
func NewSystemTaskProgressReporter(task *model.SystemTask, runnerID string) func(processed, total int) {
	const minWriteInterval = 2 * time.Second
	var (
		lastWriteAt  time.Time
		lastProgress = -1
	)
	return func(processed, total int) {
		progress := 100
		if total > 0 {
			progress = processed * 100 / total
		}
		if progress < 0 {
			progress = 0
		} else if progress > 100 {
			progress = 100
		}

		if progress < 100 {
			if progress == lastProgress {
				return
			}
			if !lastWriteAt.IsZero() && time.Since(lastWriteAt) < minWriteInterval {
				return
			}
		}
		lastProgress = progress
		lastWriteAt = time.Now()

		state := SystemTaskProgress{Total: total, Processed: processed, Progress: progress}
		_ = model.UpdateSystemTaskState(task.TaskID, runnerID, state)
	}
}
