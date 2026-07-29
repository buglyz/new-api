package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestRetryParamWithinFailoverBudget(t *testing.T) {
	previous := common.RelayFailoverBudget
	common.RelayFailoverBudget = 1
	t.Cleanup(func() { common.RelayFailoverBudget = previous })

	assert.True(t, (&RetryParam{StartedAt: time.Now()}).WithinFailoverBudget())
	assert.False(t, (&RetryParam{StartedAt: time.Now().Add(-2 * time.Second)}).WithinFailoverBudget())
	assert.True(t, (&RetryParam{}).WithinFailoverBudget())
}
