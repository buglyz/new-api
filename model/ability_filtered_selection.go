package model

import (
	"sort"

	"github.com/QuantumNous/new-api/common"
)

// getChannelWithFilter applies the personal circuit filter before choosing a
// priority. This matters when the memory channel cache is disabled: an open
// high-priority circuit must not hide a healthy lower-priority candidate.
func getChannelWithFilter(group, modelName string, retry int, requestPath string, channelFilter func(int) bool) (*Channel, error) {
	var abilities []Ability
	if err := DB.Where(commonGroupCol+" = ? and model = ? and enabled = ?", group, modelName, true).
		Order("priority DESC").
		Order("weight DESC").
		Find(&abilities).Error; err != nil {
		return nil, err
	}
	abilities = filterAbilitiesByRequestPathAndModel(abilities, requestPath, modelName)
	priorities := make([]int64, 0)
	seenPriorities := map[int64]struct{}{}
	for _, ability := range abilities {
		priority := int64(0)
		if ability.Priority != nil {
			priority = *ability.Priority
		}
		if _, exists := seenPriorities[priority]; !exists {
			seenPriorities[priority] = struct{}{}
			priorities = append(priorities, priority)
		}
	}
	sort.Slice(priorities, func(i, j int) bool { return priorities[i] > priorities[j] })
	if retry >= len(priorities) {
		retry = len(priorities) - 1
	}
	for priorityIndex := retry; priorityIndex < len(priorities); priorityIndex++ {
		targetPriority := priorities[priorityIndex]
		targets := make([]Ability, 0, len(abilities))
		weightSum := 0
		for _, ability := range abilities {
			priority := int64(0)
			if ability.Priority != nil {
				priority = *ability.Priority
			}
			if priority == targetPriority && channelFilter(ability.ChannelId) {
				targets = append(targets, ability)
				weightSum += int(ability.Weight) + 10
			}
		}
		if len(targets) == 0 {
			continue
		}

		weight := common.GetRandomInt(weightSum)
		channelID := targets[len(targets)-1].ChannelId
		for _, ability := range targets {
			weight -= int(ability.Weight) + 10
			if weight <= 0 {
				channelID = ability.ChannelId
				break
			}
		}
		var channel Channel
		if err := DB.First(&channel, "id = ?", channelID).Error; err != nil {
			return nil, err
		}
		return &channel, nil
	}
	return nil, nil
}
