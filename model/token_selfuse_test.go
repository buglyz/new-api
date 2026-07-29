package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestValidateUserTokenIgnoresLegacyExhaustedQuota(t *testing.T) {
	previousDB := DB
	previousRedis := common.RedisEnabled
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Token{}))
	DB = db
	common.RedisEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		common.RedisEnabled = previousRedis
		common.SetMainDatabaseType(previousType)
		initCol()
	})

	token := &Token{
		UserId: 1, Key: "exhausted-key", Name: "legacy", Status: common.TokenStatusExhausted,
		RemainQuota: 0, UnlimitedQuota: false,
	}
	require.NoError(t, db.Create(token).Error)

	validated, err := ValidateUserToken(token.Key)
	require.NoError(t, err)
	assert.Equal(t, token.Id, validated.Id)
}
