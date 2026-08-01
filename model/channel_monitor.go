package model

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ChannelMonitorStatusSuccess   = "success"
	ChannelMonitorStatusFailure   = "failure"
	ChannelMonitorHealthHealthy   = "healthy"
	ChannelMonitorHealthDegraded  = "degraded"
	ChannelMonitorHealthDown      = "down"
	channelMonitorDeleteBatchSize = 100
	channelMonitorWindowSeconds   = 24 * 60 * 60
)

type ChannelMonitorResult struct {
	ID           int64  `json:"id" gorm:"primaryKey"`
	ChannelID    int    `json:"channel_id" gorm:"index:idx_channel_monitor_target_time,priority:1"`
	ChannelName  string `json:"channel_name" gorm:"type:varchar(255)"`
	Groups       string `json:"groups" gorm:"type:text"`
	Model        string `json:"model" gorm:"type:varchar(255);index:idx_channel_monitor_target_time,priority:2"`
	Status       string `json:"status" gorm:"type:varchar(16);index"`
	Health       string `json:"health" gorm:"type:varchar(16);index"`
	StateChanged bool   `json:"state_changed"`
	Attempts     int    `json:"attempts"`
	LatencyMS    int64  `json:"latency_ms"`
	HTTPStatus   int    `json:"http_status"`
	Error        string `json:"error" gorm:"type:text"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index:idx_channel_monitor_target_time,priority:3"`
}

type ChannelMonitorTarget struct {
	ChannelID      int     `json:"channel_id"`
	ChannelName    string  `json:"channel_name"`
	Groups         string  `json:"groups"`
	Model          string  `json:"model"`
	Status         string  `json:"status"`
	Health         string  `json:"health"`
	StateChanged   bool    `json:"state_changed"`
	Attempts       int     `json:"attempts"`
	LatencyMS      int64   `json:"latency_ms"`
	HTTPStatus     int     `json:"http_status"`
	Error          string  `json:"error"`
	CreatedAt      int64   `json:"created_at"`
	SuccessRate24H float64 `json:"success_rate_24h"`
	Samples24H     int64   `json:"samples_24h"`
}

type channelMonitorStats struct {
	ChannelID int
	Model     string
	Total     int64
	Succeeded int64
}

type ChannelMonitorTargetRef struct {
	ChannelID int
	Model     string
}

func (result *ChannelMonitorResult) BeforeCreate(_ *gorm.DB) error {
	if result.CreatedAt == 0 {
		result.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func CreateChannelMonitorResult(result ChannelMonitorResult, failureThreshold, historyLimit int) (*ChannelMonitorResult, error) {
	if result.ChannelID <= 0 || strings.TrimSpace(result.Model) == "" {
		return nil, errors.New("channel monitor result requires a channel and model")
	}
	if result.Status != ChannelMonitorStatusSuccess && result.Status != ChannelMonitorStatusFailure {
		return nil, errors.New("invalid channel monitor result status")
	}
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	if historyLimit < 1 {
		historyLimit = 1
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var latest ChannelMonitorResult
		latestErr := tx.Where("channel_id = ? AND model = ?", result.ChannelID, result.Model).Order("id desc").First(&latest).Error
		if latestErr != nil && !errors.Is(latestErr, gorm.ErrRecordNotFound) {
			return latestErr
		}
		result.Health = monitorResultHealth(tx, result, failureThreshold)
		result.StateChanged = latestErr == nil && latest.Health != result.Health
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&ChannelMonitorResult{}).Where("channel_id = ? AND model = ?", result.ChannelID, result.Model).Count(&count).Error; err != nil {
			return err
		}
		staleCount := int(count) - historyLimit
		if staleCount <= 0 {
			return nil
		}
		var staleIDs []int64
		cutoff := common.GetTimestamp() - channelMonitorWindowSeconds
		if err := tx.Model(&ChannelMonitorResult{}).
			Where("channel_id = ? AND model = ? AND created_at < ?", result.ChannelID, result.Model, cutoff).
			Order("created_at asc, id asc").Limit(staleCount).Pluck("id", &staleIDs).Error; err != nil {
			return err
		}
		if len(staleIDs) > 0 {
			return tx.Where("id IN ?", staleIDs).Delete(&ChannelMonitorResult{}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func monitorResultHealth(tx *gorm.DB, result ChannelMonitorResult, threshold int) string {
	if result.Status == ChannelMonitorStatusSuccess {
		return ChannelMonitorHealthHealthy
	}
	if threshold == 1 {
		return ChannelMonitorHealthDown
	}
	var recent []ChannelMonitorResult
	if err := tx.Where("channel_id = ? AND model = ?", result.ChannelID, result.Model).Order("id desc").Limit(threshold - 1).Find(&recent).Error; err != nil {
		return ChannelMonitorHealthDegraded
	}
	failures := 1
	for _, previous := range recent {
		if previous.Status != ChannelMonitorStatusFailure {
			break
		}
		failures++
	}
	if failures >= threshold {
		return ChannelMonitorHealthDown
	}
	return ChannelMonitorHealthDegraded
}

func ListChannelMonitorTargets(since int64) ([]ChannelMonitorTarget, error) {
	latestRows := DB.Model(&ChannelMonitorResult{}).Select("MAX(id)").Group("channel_id, model")
	var latest []ChannelMonitorResult
	if err := DB.Where("id IN (?)", latestRows).Order("health desc, id desc").Find(&latest).Error; err != nil {
		return nil, err
	}
	statsByTarget := map[string]channelMonitorStats{}
	var stats []channelMonitorStats
	if since > 0 {
		if err := DB.Model(&ChannelMonitorResult{}).Select("channel_id, model, COUNT(*) AS total, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS succeeded", ChannelMonitorStatusSuccess).Where("created_at >= ?", since).Group("channel_id, model").Scan(&stats).Error; err != nil {
			return nil, err
		}
		for _, stat := range stats {
			statsByTarget[channelMonitorTargetKey(stat.ChannelID, stat.Model)] = stat
		}
	}
	targets := make([]ChannelMonitorTarget, 0, len(latest))
	for _, result := range latest {
		key := channelMonitorTargetKey(result.ChannelID, result.Model)
		stat := statsByTarget[key]
		target := ChannelMonitorTarget{ChannelID: result.ChannelID, ChannelName: result.ChannelName, Groups: result.Groups, Model: result.Model, Status: result.Status, Health: result.Health, StateChanged: result.StateChanged, Attempts: result.Attempts, LatencyMS: result.LatencyMS, HTTPStatus: result.HTTPStatus, Error: result.Error, CreatedAt: result.CreatedAt, Samples24H: stat.Total}
		if stat.Total > 0 {
			target.SuccessRate24H = float64(stat.Succeeded) / float64(stat.Total)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func ListChannelMonitorHistory(channelID int, modelName string, limit int) ([]ChannelMonitorResult, error) {
	if limit < 1 {
		limit = 60
	}
	var results []ChannelMonitorResult
	err := DB.Where("channel_id = ? AND model = ?", channelID, modelName).Order("created_at desc, id desc").Limit(limit).Find(&results).Error
	return results, err
}

func DeleteStaleChannelMonitorTargets(known []ChannelMonitorTargetRef) error {
	knownKeys := make(map[string]struct{}, len(known))
	for _, target := range known {
		knownKeys[channelMonitorTargetKey(target.ChannelID, target.Model)] = struct{}{}
	}
	var stored []channelMonitorStats
	if err := DB.Model(&ChannelMonitorResult{}).Select("channel_id, model").Group("channel_id, model").Scan(&stored).Error; err != nil {
		return err
	}
	stale := make([]channelMonitorStats, 0)
	for _, target := range stored {
		if _, ok := knownKeys[channelMonitorTargetKey(target.ChannelID, target.Model)]; ok {
			continue
		}
		stale = append(stale, target)
	}
	for start := 0; start < len(stale); start += channelMonitorDeleteBatchSize {
		end := min(start+channelMonitorDeleteBatchSize, len(stale))
		query := DB.Where("1 = 0")
		for _, target := range stale[start:end] {
			query = query.Or("channel_id = ? AND model = ?", target.ChannelID, target.Model)
		}
		if err := query.Delete(&ChannelMonitorResult{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func channelMonitorTargetKey(channelID int, modelName string) string {
	return strconv.Itoa(channelID) + "#" + strings.TrimSpace(modelName)
}
