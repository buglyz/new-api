package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// CancelSystemTasks records a shared cancellation intent. Pending rows become
// terminal immediately; running rows keep their lease until their handler
// observes the intent and finishes normally.
func CancelSystemTasks(taskType, reason string) error {
	if DB == nil {
		return nil
	}
	if reason == "" {
		reason = "task canceled"
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&SystemTask{}).
			Where("type = ? AND status = ?", taskType, SystemTaskStatusPending).
			Updates(map[string]any{
				"status":     SystemTaskStatusFailed,
				"active_key": nil,
				"error":      reason,
				"updated_at": common.GetTimestamp(),
			}).Error; err != nil {
			return err
		}
		return tx.Model(&SystemTask{}).
			Where("type = ? AND status = ?", taskType, SystemTaskStatusRunning).
			Updates(map[string]any{
				"cancel_requested": true,
				"error":            reason,
				"updated_at":       common.GetTimestamp(),
			}).Error
	})
}

func IsSystemTaskCancellationRequested(taskID string) (bool, error) {
	if DB == nil || taskID == "" {
		return false, nil
	}
	var task SystemTask
	if err := DB.Select("cancel_requested").Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return false, err
	}
	return task.CancelRequested, nil
}
