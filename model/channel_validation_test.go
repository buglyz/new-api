package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelsExistRequiresEveryRequestedChannel(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&Channel{}))
	require.NoError(t, db.Create(&[]Channel{
		{Id: 1, Name: "primary", Key: "credential-a"},
		{Id: 2, Name: "backup", Key: "credential-b"},
	}).Error)

	exists, err := ChannelsExist([]int{1, 2})
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = ChannelsExist([]int{1, 3})
	require.NoError(t, err)
	assert.False(t, exists)
}
