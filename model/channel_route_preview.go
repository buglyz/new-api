package model

import (
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type ChannelRouteCandidate struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Priority    int64  `json:"priority"`
	Weight      uint   `json:"weight"`
}

// GetChannelRouteCandidates returns the real enabled candidates used by one
// explicit group/model pair. It intentionally does not perform weighted random
// selection, so callers can present a read-only route preview without changing
// routing state or promising a deterministic winner.
func GetChannelRouteCandidates(group, modelName, requestPath string) ([]ChannelRouteCandidate, error) {
	abilities, err := getRoutePreviewAbilities(group, modelName)
	if err != nil {
		return nil, err
	}
	if len(abilities) == 0 {
		normalized := ratio_setting.FormatMatchingModelName(modelName)
		if normalized != "" && normalized != modelName {
			abilities, err = getRoutePreviewAbilities(group, normalized)
			if err != nil {
				return nil, err
			}
		}
	}
	abilities = filterAbilitiesByRequestPathAndModel(abilities, requestPath, modelName)
	if len(abilities) == 0 {
		return []ChannelRouteCandidate{}, nil
	}

	channelIDs := make([]int, 0, len(abilities))
	for _, ability := range abilities {
		channelIDs = append(channelIDs, ability.ChannelId)
	}
	var channels []struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := DB.Model(&Channel{}).Select("id, name").Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
		return nil, err
	}
	names := make(map[int]string, len(channels))
	for _, channel := range channels {
		names[channel.ID] = channel.Name
	}

	result := make([]ChannelRouteCandidate, 0, len(abilities))
	for _, ability := range abilities {
		priority := int64(0)
		if ability.Priority != nil {
			priority = *ability.Priority
		}
		result = append(result, ChannelRouteCandidate{
			ChannelID: ability.ChannelId, ChannelName: names[ability.ChannelId],
			Priority: priority, Weight: ability.Weight,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}
		return result[i].ChannelID < result[j].ChannelID
	})
	return result, nil
}

func getRoutePreviewAbilities(group, modelName string) ([]Ability, error) {
	var abilities []Ability
	err := DB.Table("abilities").
		Select("abilities.*").
		Joins("JOIN channels ON abilities.channel_id = channels.id").
		Where("abilities."+commonGroupCol+" = ? AND abilities.model = ? AND abilities.enabled = ?", group, modelName, true).
		Where("channels.status = ?", common.ChannelStatusEnabled).
		Find(&abilities).Error
	return abilities, err
}
