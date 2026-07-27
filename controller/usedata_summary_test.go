package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetUserUsageSummaryRestrictsTotalsToAuthenticatedUser(t *testing.T) {
	setupFlowControllerTestDB(t)
	require.NoError(t, model.DB.Create(&model.QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 1300,
		Count:     4,
		Quota:     9999,
		TokenUsed: 60,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/self/summary", nil)

	GetUserUsageSummary(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool                   `json:"success"`
		Data    model.UserUsageSummary `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.EqualValues(t, 100, payload.Data.TotalTokens)
}
