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

type usageSummaryResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    model.UserUsageSummary `json:"data"`
}

func requestUsageSummary(t *testing.T, userId int) usageSummaryResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", userId)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/self/summary", nil)

	GetUserUsageSummary(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload usageSummaryResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload
}

// resetQuotaDataCache isolates the process-wide flush cache so summary tests do
// not leak pending tokens into each other.
func resetQuotaDataCache(t *testing.T) {
	t.Helper()
	originalDataExportEnabled := common.DataExportEnabled
	common.DataExportEnabled = true
	model.CacheQuotaDataLock.Lock()
	model.CacheQuotaData = make(map[string]*model.QuotaData)
	model.CacheQuotaDataLock.Unlock()
	t.Cleanup(func() {
		common.DataExportEnabled = originalDataExportEnabled
		model.CacheQuotaDataLock.Lock()
		model.CacheQuotaData = make(map[string]*model.QuotaData)
		model.CacheQuotaDataLock.Unlock()
	})
}

func TestGetUserUsageSummaryRestrictsTotalsToAuthenticatedUser(t *testing.T) {
	setupFlowControllerTestDB(t)
	resetQuotaDataCache(t)
	require.NoError(t, model.DB.Create(&model.QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: 1300,
		Count:     4,
		Quota:     9999,
		TokenUsed: 60,
	}).Error)

	payload := requestUsageSummary(t, 1)

	require.True(t, payload.Success)
	require.EqualValues(t, 100, payload.Data.TotalTokens)
}

func TestGetUserUsageSummaryIncludesUnflushedCachedTokens(t *testing.T) {
	setupFlowControllerTestDB(t)
	resetQuotaDataCache(t)
	now := common.GetTimestamp()

	// Usage recorded after the last flush lives only in the in-memory cache.
	// The summary must not lag a whole export interval behind it.
	model.LogQuotaData(model.QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: now,
		Quota:     5,
		TokenUsed: 25,
	})

	payload := requestUsageSummary(t, 1)

	require.True(t, payload.Success)
	require.EqualValues(t, 65, payload.Data.TotalTokens)
	require.EqualValues(t, 25, payload.Data.Last24hTokens)
}

func TestGetUserUsageSummaryExcludesOtherUsersCachedTokens(t *testing.T) {
	setupFlowControllerTestDB(t)
	resetQuotaDataCache(t)
	model.LogQuotaData(model.QuotaDataLogParams{
		UserID:    2,
		Username:  "bob",
		ModelName: "gpt-b",
		CreatedAt: 7200,
		TokenUsed: 500,
	})

	payload := requestUsageSummary(t, 1)

	require.True(t, payload.Success)
	require.EqualValues(t, 40, payload.Data.TotalTokens)
}

func TestGetUserUsageSummaryDoesNotDoubleCountFlushedCache(t *testing.T) {
	setupFlowControllerTestDB(t)
	resetQuotaDataCache(t)
	now := common.GetTimestamp()
	model.LogQuotaData(model.QuotaDataLogParams{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-a",
		CreatedAt: now,
		TokenUsed: 25,
	})
	model.SaveQuotaDataCache()

	payload := requestUsageSummary(t, 1)

	require.True(t, payload.Success)
	require.EqualValues(t, 65, payload.Data.TotalTokens)
	require.EqualValues(t, 25, payload.Data.Last24hTokens)
}

func TestGetUserUsageSummaryReportsTrackingDisabled(t *testing.T) {
	setupFlowControllerTestDB(t)
	resetQuotaDataCache(t)
	common.DataExportEnabled = false

	payload := requestUsageSummary(t, 1)

	require.True(t, payload.Success)
	require.False(t, payload.Data.TokensTracked)
	require.EqualValues(t, 40, payload.Data.TotalTokens)
}

func TestGetUserUsageSummaryEncodesTokenTotalsAsExactStrings(t *testing.T) {
	setupFlowControllerTestDB(t)
	resetQuotaDataCache(t)
	require.NoError(t, model.DB.Create(&model.QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-large",
		CreatedAt: common.GetTimestamp(),
		TokenUsed: 9_007_199_254_740_953,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/self/summary", nil)

	GetUserUsageSummary(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"total_tokens":"9007199254740993"`)
	require.Contains(t, recorder.Body.String(), `"last_24h_tokens":"9007199254740953"`)
}
