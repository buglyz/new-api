package service

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	model.DB = db
	model.LOG_DB = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	if err := db.AutoMigrate(
		&model.Task{},
		&model.User{},
		&model.Token{},
		&model.Log{},
		&model.Channel{},
		&model.TopUp{},
		&model.UserSubscription{},
		&model.SystemTask{},
		&model.SystemTaskLock{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}
	os.Exit(m.Run())
}

func resetUsageTables(t *testing.T) {
	t.Helper()
	for _, table := range []string{"tasks", "users", "tokens", "logs", "channels", "user_subscriptions"} {
		require.NoError(t, model.DB.Exec("DELETE FROM "+table).Error)
	}
}

func truncate(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		for _, table := range []string{
			"tasks", "users", "tokens", "logs", "channels", "top_ups",
			"user_subscriptions", "system_task_locks", "system_tasks",
		} {
			_ = model.DB.Exec("DELETE FROM " + table).Error
		}
	})
}

func seedUser(t *testing.T, id int, quota int) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.User{
		Id: id, Username: "test_user", Quota: quota, Status: common.UserStatusEnabled,
	}).Error)
}

func seedToken(t *testing.T, id, userID int, key string, remainQuota int) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Token{
		Id: id, UserId: userID, Key: key, Name: "test_token",
		Status: common.TokenStatusEnabled, RemainQuota: remainQuota,
	}).Error)
}

func makeTask(userID, channelID, quota, tokenID int, billingSource string, subscriptionID int) *model.Task {
	return &model.Task{
		TaskID: "task_" + time.Now().Format("150405.000"), UserId: userID,
		ChannelId: channelID, Quota: quota, Status: model.TaskStatus(model.TaskStatusInProgress),
		Group: "default", CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
		Properties: model.Properties{OriginModelName: "test-model"},
		PrivateData: model.TaskPrivateData{
			BillingSource: billingSource, SubscriptionId: subscriptionID, TokenId: tokenID,
		},
	}
}

func getUserQuota(t *testing.T, id int) int {
	t.Helper()
	var user model.User
	require.NoError(t, model.DB.Select("quota").First(&user, id).Error)
	return user.Quota
}

func getTokenRemainQuota(t *testing.T, id int) int {
	t.Helper()
	var token model.Token
	require.NoError(t, model.DB.Select("remain_quota").First(&token, id).Error)
	return token.RemainQuota
}

func getTaskQuota(t *testing.T, id int64) int {
	t.Helper()
	var task model.Task
	require.NoError(t, model.DB.Select("quota").First(&task, id).Error)
	return task.Quota
}

func countLogs(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Count(&count).Error)
	return count
}

func seedLegacyBillingState(t *testing.T) *model.Task {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.User{
		Id: 1, Username: "self", Quota: 900, UsedQuota: 100, Status: common.UserStatusEnabled,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Token{
		Id: 2, UserId: 1, Key: "legacy", Name: "legacy", Status: common.TokenStatusEnabled,
		RemainQuota: 700, UsedQuota: 50,
	}).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id: 3, UserId: 1, AmountTotal: 1000, AmountUsed: 250, Status: "active",
		StartTime: time.Now().Add(-time.Hour).Unix(), EndTime: time.Now().Add(time.Hour).Unix(),
	}).Error)
	task := &model.Task{
		TaskID: "task_legacy", UserId: 1, ChannelId: 4, Quota: 120,
		Status: model.TaskStatus(model.TaskStatusInProgress), CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{TokenId: 2, BillingSource: BillingSourceSubscription, SubscriptionId: 3},
	}
	require.NoError(t, model.DB.Create(task).Error)
	return task
}

func assertLegacyBalances(t *testing.T) {
	t.Helper()
	var user model.User
	var token model.Token
	var subscription model.UserSubscription
	require.NoError(t, model.DB.First(&user, 1).Error)
	require.NoError(t, model.DB.First(&token, 2).Error)
	require.NoError(t, model.DB.First(&subscription, 3).Error)
	assert.Equal(t, 900, user.Quota)
	assert.Equal(t, 100, user.UsedQuota)
	assert.Equal(t, 700, token.RemainQuota)
	assert.Equal(t, 50, token.UsedQuota)
	assert.Equal(t, int64(250), subscription.AmountUsed)
}

func TestRefundTaskQuotaClearsLegacyMarkerWithoutChangingBalances(t *testing.T) {
	resetUsageTables(t)
	task := seedLegacyBillingState(t)

	require.True(t, RefundTaskQuota(context.Background(), task, "failed"))

	var saved model.Task
	require.NoError(t, model.DB.First(&saved, task.ID).Error)
	assert.Zero(t, saved.Quota)
	assertLegacyBalances(t)
}

func TestRecalculateTaskQuotaDoesNotChangeLegacyAmounts(t *testing.T) {
	resetUsageTables(t)
	task := seedLegacyBillingState(t)

	RecalculateTaskQuota(context.Background(), task, 999, "legacy callback")
	RecalculateTaskQuotaByTokens(context.Background(), task, 12345)

	var saved model.Task
	require.NoError(t, model.DB.First(&saved, task.ID).Error)
	assert.Equal(t, 120, saved.Quota)
	assertLegacyBalances(t)
}

func TestLogTaskConsumptionRecordsUsageWithoutAmount(t *testing.T) {
	resetUsageTables(t)
	require.NoError(t, model.DB.Create(&model.User{
		Id: 1, Username: "self", Status: common.UserStatusEnabled,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id: 4, Name: "upstream", Key: "secret", Status: common.ChannelStatusEnabled,
	}).Error)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/videos", nil)
	ctx.Set("token_name", "personal")
	info := &relaycommon.RelayInfo{
		UserId: 1, TokenId: 2, UsingGroup: "default", OriginModelName: "video-model",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 4, UpstreamModelName: "mapped-video-model", IsModelMapped: true,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{Action: "generate"},
	}

	LogTaskConsumption(ctx, info)

	var log model.Log
	require.NoError(t, model.DB.Last(&log).Error)
	assert.Zero(t, log.Quota)
	assert.Equal(t, "video-model", log.ModelName)
	var user model.User
	require.NoError(t, model.DB.First(&user, 1).Error)
	assert.Equal(t, 1, user.RequestCount)
}
