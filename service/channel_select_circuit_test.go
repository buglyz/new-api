package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelSelectionHonorsCircuitCooldownAndSingleHalfOpenClaim(t *testing.T) {
	previousMode := operation_setting.SelfUseModeEnabled
	previousCache := common.MemoryCacheEnabled
	previousCircuits := personalCircuits
	now := time.Unix(1_700_000_000, 0)
	operation_setting.SelfUseModeEnabled = true
	common.MemoryCacheEnabled = true
	personalCircuits = newPersonalCircuitManager(func() time.Time { return now })
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousMode
		common.MemoryCacheEnabled = previousCache
		personalCircuits = previousCircuits
	})

	require.NoError(t, model.DB.AutoMigrate(&model.Ability{}))
	require.NoError(t, model.DB.Where("1 = 1").Delete(&model.Ability{}).Error)
	require.NoError(t, model.DB.Where("1 = 1").Delete(&model.Channel{}).Error)
	t.Cleanup(func() {
		_ = model.DB.Where("1 = 1").Delete(&model.Ability{}).Error
		_ = model.DB.Where("1 = 1").Delete(&model.Channel{}).Error
	})

	priority := int64(0)
	weight := uint(0)
	channel := model.Channel{
		Name: "cooling", Status: common.ChannelStatusEnabled,
		Models: "test-model", Group: "default", Priority: &priority, Weight: &weight,
	}
	require.NoError(t, model.DB.Create(&channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group: "default", Model: "test-model", ChannelId: channel.Id,
		Enabled: true, Priority: &priority,
	}).Error)
	model.InitChannelCache()
	t.Cleanup(model.InitChannelCache)

	recordFailures(personalCircuits, personalCircuitFailureThreshold, channel.Id, channel.Name, "test-model", RelayAttempt{
		Outcome: RelayAttemptUpstream5xx, StatusCode: 503,
	})
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	param := &RetryParam{Ctx: ctx, TokenGroup: "default", ModelName: "test-model"}

	selected, err := getRandomSatisfiedChannel(param, "default", 0)
	require.NoError(t, err)
	assert.Nil(t, selected, "cooldown must not be bypassed")

	now = now.Add(personalCircuitBaseBackoff)
	selected, err = getRandomSatisfiedChannel(param, "default", 0)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, channel.Id, selected.Id)

	selected, err = getRandomSatisfiedChannel(param, "default", 0)
	require.NoError(t, err)
	assert.Nil(t, selected, "only one half-open probe may hold the lease")
}
