package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetPricingUsesUnifiedSuccessResponse(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	user := &model.User{Id: 1, Username: "owner", Group: "default", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)

	GetPricing(ctx)

	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])
	require.Equal(t, "", response["message"])
	require.NotNil(t, response["data"])
	require.NotContains(t, response, "vendors")

	payload, ok := response["data"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, payload, "models")
	require.Contains(t, payload, "vendors")
	require.Equal(t, "self-use-model-square-v1", payload["pricing_version"])
}
