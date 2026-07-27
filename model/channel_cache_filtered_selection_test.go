package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilteredCachedChannelPreservesOriginalPriorityLadder(t *testing.T) {
	previousChannels := channelsIDM
	high, low, last := int64(10), int64(0), int64(-10)
	zero := uint(0)
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Priority: &high, Weight: &zero},
		2: {Id: 2, Priority: &low, Weight: &zero},
		3: {Id: 3, Priority: &last, Weight: &zero},
	}
	t.Cleanup(func() { channelsIDM = previousChannels })

	selected, err := getFilteredCachedChannel([]int{1, 2, 3}, 1, func(channelID int) bool {
		return channelID != 1
	})
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 2, selected.Id)
}
