package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const personalReliabilityBatchLimit = 100

type personalReliabilityChannelRequest struct {
	ChannelIDs []int `json:"channel_ids"`
}

type personalRoutePreviewRequest struct {
	Group       string `json:"group"`
	Model       string `json:"model"`
	RequestPath string `json:"request_path"`
}

type personalRoutePreviewCandidate struct {
	model.ChannelRouteCandidate
	CircuitStatus string `json:"circuit_status"`
	RetryAt       int64  `json:"retry_at,omitempty"`
	Eligible      bool   `json:"eligible"`
}

func GetPersonalReliability(c *gin.Context) {
	circuits, transitions, policy := service.GetPersonalCircuitSnapshot()
	if len(transitions) > 20 {
		transitions = transitions[:20]
	}
	common.ApiSuccess(c, gin.H{
		"circuits":    circuits,
		"transitions": transitions,
		"policy":      policy,
	})
}

func ProbePersonalReliabilityChannels(c *gin.Context) {
	enqueuePersonalReliabilityTest(c, false)
}

func RecoverPersonalReliabilityChannels(c *gin.Context) {
	enqueuePersonalReliabilityTest(c, true)
}

func ResetPersonalReliabilityCircuits(c *gin.Context) {
	channelIDs, ok := parsePersonalReliabilityChannelIDs(c)
	if !ok {
		return
	}
	common.ApiSuccess(c, gin.H{"reset_count": service.ResetPersonalCircuits(channelIDs)})
}

func enqueuePersonalReliabilityTest(c *gin.Context, resetCircuitsOnSuccess bool) {
	channelIDs, ok := parsePersonalReliabilityChannelIDs(c)
	if !ok {
		return
	}
	task, created, err := service.EnqueueSystemTask(model.SystemTaskTypeChannelTest, channelTestTaskPayload{
		Mode:                   operation_setting.ChannelTestModeScheduledAll,
		ChannelIDs:             channelIDs,
		ResetCircuitsOnSuccess: resetCircuitsOnSuccess,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !created {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgPersonalReliabilityTaskActive),
			"data":    gin.H{"task_id": task.TaskID, "status": task.Status, "type": task.Type},
		})
		return
	}
	common.ApiSuccess(c, gin.H{"task_id": task.TaskID, "status": task.Status})
}

func parsePersonalReliabilityChannelIDs(c *gin.Context) ([]int, bool) {
	request := personalReliabilityChannelRequest{}
	if err := common.UnmarshalBodyReusable(c, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityInvalidPayload)
		return nil, false
	}
	seen := make(map[int]struct{}, len(request.ChannelIDs))
	channelIDs := make([]int, 0, len(request.ChannelIDs))
	for _, channelID := range request.ChannelIDs {
		if channelID <= 0 {
			common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityInvalidPayload)
			return nil, false
		}
		if _, exists := seen[channelID]; exists {
			continue
		}
		seen[channelID] = struct{}{}
		channelIDs = append(channelIDs, channelID)
	}
	if len(channelIDs) == 0 {
		common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityChannelRequired)
		return nil, false
	}
	if len(channelIDs) > personalReliabilityBatchLimit {
		common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityBatchLimit, map[string]any{"Limit": personalReliabilityBatchLimit})
		return nil, false
	}
	channelsExist, err := model.ChannelsExist(channelIDs)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if !channelsExist {
		common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityChannelNotFound)
		return nil, false
	}
	return channelIDs, true
}

func SimulatePersonalRoute(c *gin.Context) {
	request := personalRoutePreviewRequest{}
	if err := common.UnmarshalBodyReusable(c, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityInvalidPayload)
		return
	}
	request.Group = strings.TrimSpace(request.Group)
	request.Model = strings.TrimSpace(request.Model)
	request.RequestPath = strings.TrimSpace(request.RequestPath)
	if request.Group == "" || request.Model == "" || len(request.Group) > 64 || len(request.Model) > 255 || len(request.RequestPath) > 512 {
		common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityGroupModelRequired)
		return
	}
	if request.Group == "auto" {
		common.ApiErrorI18n(c, i18n.MsgPersonalReliabilityExplicitGroup)
		return
	}

	candidates, err := model.GetChannelRouteCandidates(request.Group, request.Model, request.RequestPath)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	circuits, _, _ := service.GetPersonalCircuitSnapshot()
	circuitByChannel := personalCircuitPreviewIndex(circuits, request.Model)
	preview := make([]personalRoutePreviewCandidate, 0, len(candidates))
	var highestAvailablePriority *int64
	for _, candidate := range candidates {
		item := personalRoutePreviewCandidate{
			ChannelRouteCandidate: candidate,
			CircuitStatus:         string(service.PersonalCircuitClosed),
			Eligible:              true,
		}
		if circuit, exists := circuitByChannel[candidate.ChannelID]; exists {
			item.CircuitStatus = string(circuit.Status)
			item.RetryAt = circuit.RetryAt
			item.Eligible = service.PersonalCircuitCanAttempt(candidate.ChannelID, request.Model)
		}
		if item.Eligible && (highestAvailablePriority == nil || candidate.Priority > *highestAvailablePriority) {
			priority := candidate.Priority
			highestAvailablePriority = &priority
		}
		preview = append(preview, item)
	}
	common.ApiSuccess(c, gin.H{
		"group": request.Group, "model": request.Model, "request_path": request.RequestPath,
		"strategy": "priority_then_weighted_random", "highest_available_priority": highestAvailablePriority,
		"candidates": preview,
	})
}

func personalCircuitPreviewIndex(circuits []service.PersonalCircuit, modelName string) map[int]service.PersonalCircuit {
	index := make(map[int]service.PersonalCircuit)
	for _, circuit := range circuits {
		if circuit.Model == modelName {
			index[circuit.ChannelID] = circuit
		}
	}
	for _, circuit := range circuits {
		if circuit.Model == "*" {
			if _, exists := index[circuit.ChannelID]; !exists {
				index[circuit.ChannelID] = circuit
			}
		}
	}
	return index
}
