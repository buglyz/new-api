package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if common.StringsContains(item.EnableGroup, "all") {
			filtered = append(filtered, item)
			continue
		}
		for _, group := range item.EnableGroup {
			if _, ok := usableGroup[group]; ok {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

// GetPricing returns the authenticated owner's read-only model catalog.
// Price fields are reference metadata for the model square; relay admission
// and settlement do not consume them in the self-use build.
func GetPricing(c *gin.Context) {
	user, err := model.GetUserCache(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	usableGroup := service.GetUserUsableGroups(user.Group)
	pricing := filterPricingByUsableGroups(model.GetPricing(), usableGroup)
	groupRatio := make(map[string]float64, len(usableGroup))
	for group := range usableGroup {
		ratio := ratio_setting.GetGroupRatio(group)
		if userRatio, ok := ratio_setting.GetGroupGroupRatio(user.Group, group); ok {
			ratio = userRatio
		}
		groupRatio[group] = ratio
	}

	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        service.GetUserAutoGroup(user.Group),
		"pricing_version":    "self-use-model-square-v1",
	})
}
