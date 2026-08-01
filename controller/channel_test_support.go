package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func isChannelTestSupported(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeMidjourney,
		constant.ChannelTypeMidjourneyPlus,
		constant.ChannelTypeSunoAPI,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeVidu:
		return false
	default:
		return true
	}
}

func setChannelTestRequest(c *gin.Context, ctx context.Context, requestPath string, quiet bool) {
	if quiet {
		ctx = common.WithSuppressedRequestLogs(ctx)
	}
	c.Request = httptest.NewRequestWithContext(ctx, http.MethodPost, requestPath, nil)
	if quiet {
		common.SetContextKey(c, constant.ContextKeySuppressRequestLogs, true)
	}
}

func channelTestBadResponseError(c *gin.Context, response *http.Response, channel *model.Channel, modelName, endpointType string, logDetails bool) *types.NewAPIError {
	var err *types.NewAPIError
	if logDetails {
		err = service.RelayErrorHandler(c.Request.Context(), response, true)
	} else {
		err = service.RelayErrorHandlerQuiet(c.Request.Context(), response)
	}
	if logDetails {
		common.SysError(fmt.Sprintf(
			"channel test bad response: channel_id=%d name=%s type=%d model=%s endpoint_type=%s status=%d err=%v",
			channel.Id, channel.Name, channel.Type, modelName, endpointType, response.StatusCode, err,
		))
	}
	return err
}

func newChannelTestResponsesRequest(modelName string, isStream, quiet bool) *dto.OpenAIResponsesRequest {
	request := &dto.OpenAIResponsesRequest{
		Model:  modelName,
		Input:  json.RawMessage(`[{"role":"user","content":"hi"}]`),
		Stream: lo.ToPtr(isStream),
	}
	if quiet {
		request.MaxOutputTokens = lo.ToPtr(uint(16))
	}
	return request
}

func configureChannelTestTokenLimit(request *dto.GeneralOpenAIRequest, modelName string, quiet bool) {
	if quiet {
		if dto.IsOpenAIReasoningOModel(modelName) {
			request.MaxCompletionTokens = lo.ToPtr(uint(16))
		} else {
			request.MaxTokens = lo.ToPtr(uint(16))
		}
		return
	}
	if dto.IsOpenAIReasoningOModel(modelName) {
		request.MaxCompletionTokens = lo.ToPtr(uint(16))
	} else if strings.Contains(modelName, "thinking") {
		if !strings.Contains(modelName, "claude") {
			request.MaxTokens = lo.ToPtr(uint(50))
		}
	} else if strings.Contains(modelName, "gemini") {
		request.MaxTokens = lo.ToPtr(uint(3000))
	} else {
		request.MaxTokens = lo.ToPtr(uint(16))
	}
}
