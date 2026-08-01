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

var (
	systemTaskRunnerMu     sync.Mutex
	systemTaskRunnerCtx    context.Context
	systemTaskRunnerCancel context.CancelFunc
	systemTaskRunnerWG     sync.WaitGroup
	systemTaskRunsMu       sync.Mutex
	systemTaskRuns         = map[string]map[string]context.CancelFunc{}
)

func startSystemTaskRunner() {
	if !common.IsMasterNode {
		return
	}
	systemTaskRunnerMu.Lock()
	if systemTaskRunnerCancel != nil {
		systemTaskRunnerMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	systemTaskRunnerCtx = ctx
	systemTaskRunnerCancel = cancel
	systemTaskRunnerWG.Add(1)
	systemTaskRunnerMu.Unlock()

	runnerID := fmt.Sprintf("%s-%s", common.NodeName, common.GetRandomString(8))
	gopool.Go(func() {
		defer systemTaskRunnerWG.Done()
		runSystemTaskLoop(ctx, runnerID)
	})
}

func StopSystemTaskRunner(ctx context.Context) error {
	systemTaskRunnerMu.Lock()
	runnerCtx := systemTaskRunnerCtx
	cancel := systemTaskRunnerCancel
	systemTaskRunnerMu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	done := make(chan struct{})
	go func() {
		systemTaskRunnerWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		systemTaskRunnerMu.Lock()
		if systemTaskRunnerCtx == runnerCtx {
			systemTaskRunnerCtx = nil
			systemTaskRunnerCancel = nil
		}
		systemTaskRunnerMu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func CancelSystemTaskRunner() {
	systemTaskRunnerMu.Lock()
	cancel := systemTaskRunnerCancel
	systemTaskRunnerMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func CancelSystemTaskType(taskType string) {
	systemTaskRunsMu.Lock()
	runs := make([]context.CancelFunc, 0, len(systemTaskRuns[taskType]))
	for _, cancel := range systemTaskRuns[taskType] {
		runs = append(runs, cancel)
	}
	systemTaskRunsMu.Unlock()
	for _, cancel := range runs {
		cancel()
	}
}

func registerSystemTaskRun(taskType, taskID string, parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	systemTaskRunsMu.Lock()
	if systemTaskRuns[taskType] == nil {
		systemTaskRuns[taskType] = map[string]context.CancelFunc{}
	}
	systemTaskRuns[taskType][taskID] = cancel
	systemTaskRunsMu.Unlock()
	return ctx, cancel
}

func unregisterSystemTaskRun(taskType, taskID string) {
	systemTaskRunsMu.Lock()
	defer systemTaskRunsMu.Unlock()
	if runs := systemTaskRuns[taskType]; runs != nil {
		delete(runs, taskID)
	}
	if len(systemTaskRuns[taskType]) == 0 {
		delete(systemTaskRuns, taskType)
	}
}

func runSystemTaskLoop(ctx context.Context, runnerID string) {
	logger.LogInfo(ctx, fmt.Sprintf("system task runner started: runner=%s idle_interval=%s", runnerID, systemTaskRunnerIdleInterval))
	ticker := time.NewTicker(systemTaskRunnerIdleInterval)
	defer ticker.Stop()
	var taskWG sync.WaitGroup
	defer taskWG.Wait()
	var lastScheduler time.Time
	var lastStaleLockCleanup time.Time
	runPass := func() {
		if ctx.Err() != nil {
			return
		}
		now := time.Now()
		if now.Sub(lastStaleLockCleanup) >= systemTaskStaleLockInterval {
			lastStaleLockCleanup = now
			if err := model.ExpireStaleSystemTaskLocks(common.GetTimestamp()); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("system task stale lock cleanup failed: %v", err))
			}
		}
		if now.Sub(lastScheduler) >= systemTaskSchedulerInterval {
			lastScheduler = now
			runSystemTaskScheduler()
		}
		runSystemTaskClaimPass(ctx, runnerID, &taskWG)
	}
	runPass()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-systemTaskWakeup:
		}
		runPass()
	}
}
