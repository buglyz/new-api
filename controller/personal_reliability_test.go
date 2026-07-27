package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePersonalReliabilityChannelIDsRejectsWholeInvalidBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/channel/reliability/probe",
		strings.NewReader(`{"channel_ids":[1,0,2]}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	channelIDs, ok := parsePersonalReliabilityChannelIDs(ctx)
	assert.False(t, ok)
	assert.Nil(t, channelIDs)

	var payload struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.False(t, payload.Success)
}
