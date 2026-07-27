package model

import (
	"fmt"
	"math/rand"
	"sort"
)

// getFilteredCachedChannel preserves the original priority ladder while
// excluding temporarily unavailable channels. Caller holds channelSyncLock.
func getFilteredCachedChannel(channels []int, retry int, channelFilter func(int) bool) (*Channel, error) {
	priorities := make(map[int]bool)
	for _, channelID := range channels {
		channel, ok := channelsIDM[channelID]
		if !ok {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelID)
		}
		priorities[int(channel.GetPriority())] = true
	}
	sortedPriorities := make([]int, 0, len(priorities))
	for priority := range priorities {
		sortedPriorities = append(sortedPriorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedPriorities)))
	if len(sortedPriorities) == 0 {
		return nil, nil
	}
	if retry >= len(sortedPriorities) {
		retry = len(sortedPriorities) - 1
	}

	for priorityIndex := retry; priorityIndex < len(sortedPriorities); priorityIndex++ {
		targetPriority := int64(sortedPriorities[priorityIndex])
		targets := make([]*Channel, 0, len(channels))
		weightSum := 0
		for _, channelID := range channels {
			channel := channelsIDM[channelID]
			if channel.GetPriority() == targetPriority && channelFilter(channelID) {
				targets = append(targets, channel)
				weightSum += channel.GetWeight()
			}
		}
		if len(targets) == 0 {
			continue
		}
		smoothingFactor, smoothingAdjustment := 1, 0
		if weightSum == 0 {
			weightSum = len(targets) * 100
			smoothingAdjustment = 100
		} else if weightSum/len(targets) < 10 {
			smoothingFactor = 100
		}
		randomWeight := rand.Intn(weightSum * smoothingFactor)
		for _, channel := range targets {
			randomWeight -= channel.GetWeight()*smoothingFactor + smoothingAdjustment
			if randomWeight < 0 {
				return channel, nil
			}
		}
	}
	return nil, nil
}
