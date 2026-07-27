package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetChannelRouteCandidatesReturnsPriorityOrderedEnabledAbilities(t *testing.T) {
	previousDB := DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
	})
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))

	priorityHigh, priorityLow, priorityLast := int64(10), int64(0), int64(-10)
	require.NoError(t, db.Create(&[]Channel{
		{Id: 1, Name: "primary", Key: "masked", Status: common.ChannelStatusEnabled},
		{Id: 2, Name: "backup", Key: "masked", Status: common.ChannelStatusEnabled},
		{Id: 3, Name: "last", Key: "masked", Status: common.ChannelStatusEnabled},
	}).Error)
	require.NoError(t, db.Create(&[]Ability{
		{Group: "default", Model: "model-a", ChannelId: 2, Enabled: true, Priority: &priorityLow, Weight: 0},
		{Group: "default", Model: "model-a", ChannelId: 1, Enabled: true, Priority: &priorityHigh, Weight: 5},
		{Group: "default", Model: "model-a", ChannelId: 3, Enabled: true, Priority: &priorityLast, Weight: 9},
	}).Error)

	candidates, err := GetChannelRouteCandidates("default", "model-a", "/v1/chat/completions")
	require.NoError(t, err)
	require.Len(t, candidates, 3)
	assert.Equal(t, 1, candidates[0].ChannelID)
	assert.Equal(t, int64(10), candidates[0].Priority)
	assert.Equal(t, 2, candidates[1].ChannelID)

	selected, err := GetChannel("default", "model-a", 0, "/v1/chat/completions")
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 1, selected.Id)

	selected, err = GetChannel("default", "model-a", 0, "/v1/chat/completions", func(channelID int) bool {
		return channelID != 1
	})
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 2, selected.Id)

	selected, err = GetChannel("default", "model-a", 1, "/v1/chat/completions", func(channelID int) bool {
		return channelID != 1
	})
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 2, selected.Id)
}
