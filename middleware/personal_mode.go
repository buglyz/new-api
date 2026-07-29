package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/gin-gonic/gin"
)

const PersonalModeDisabledCode = "PERSONAL_MODE_DISABLED"

var personalModeDisabledOptionKeys = map[string]struct{}{
	"DemoSiteEnabled":             {},
	"SelfUseModeEnabled":          {},
	"PayAddress":                  {},
	"EpayId":                      {},
	"EpayKey":                     {},
	"Price":                       {},
	"MinTopUp":                    {},
	"StripeApiSecret":             {},
	"StripeWebhookSecret":         {},
	"StripePriceId":               {},
	"StripeUnitPrice":             {},
	"StripeMinTopUp":              {},
	"StripePromotionCodesEnabled": {},
	"CreemApiKey":                 {},
	"CreemProducts":               {},
	"CreemTestMode":               {},
	"CreemWebhookSecret":          {},
	"WaffoEnabled":                {},
	"WaffoApiKey":                 {},
	"WaffoPrivateKey":             {},
	"WaffoPublicCert":             {},
	"WaffoSandboxPublicCert":      {},
	"WaffoSandboxApiKey":          {},
	"WaffoSandboxPrivateKey":      {},
	"WaffoSandbox":                {},
	"WaffoMerchantId":             {},
	"WaffoCurrency":               {},
	"WaffoUnitPrice":              {},
	"WaffoMinTopUp":               {},
	"WaffoNotifyUrl":              {},
	"WaffoReturnUrl":              {},
	"WaffoSubscriptionReturnUrl":  {},
	"WaffoPayMethods":             {},
	"WaffoPancakeMerchantID":      {},
	"WaffoPancakePrivateKey":      {},
	"WaffoPancakeReturnURL":       {},
	"WaffoPancakeStoreID":         {},
	"WaffoPancakeProductID":       {},
	"WaffoPancakeUnitPrice":       {},
	"WaffoPancakeMinTopUp":        {},
	"TopupGroupRatio":             {},
	"QuotaForNewUser":             {},
	"QuotaForInviter":             {},
	"QuotaForInvitee":             {},
	"TopUpLink":                   {},
	"PayMethods":                  {},
	"RegisterEnabled":             {},
	"PasswordRegisterEnabled":     {},
	"EmailVerificationEnabled":    {},
	"GitHubOAuthEnabled":          {},
	"GitHubClientId":              {},
	"GitHubClientSecret":          {},
	"LinuxDOOAuthEnabled":         {},
	"LinuxDOClientId":             {},
	"LinuxDOClientSecret":         {},
	"LinuxDOMinimumTrustLevel":    {},
	"WeChatAuthEnabled":           {},
	"WeChatServerAddress":         {},
	"WeChatServerToken":           {},
	"WeChatAccountQRCodeImageURL": {},
	"TelegramOAuthEnabled":        {},
	"TelegramBotToken":            {},
	"TelegramBotName":             {},
	"discord.enabled":             {},
	"discord.client_id":           {},
	"discord.client_secret":       {},
	"oidc.enabled":                {},
	"oidc.client_id":              {},
	"oidc.client_secret":          {},
	"oidc.well_known":             {},
	"oidc.authorization_endpoint": {},
	"oidc.token_endpoint":         {},
	"oidc.user_info_endpoint":     {},
}

func isPersonalModeOptionDisabled(key string) bool {
	_, disabled := personalModeDisabledOptionKeys[key]
	return disabled
}

func isPersonalModeOptionRequestDisabled(c *gin.Context) bool {
	var request struct {
		Key string `json:"key"`
	}
	if err := common.UnmarshalBodyReusable(c, &request); err != nil {
		return false
	}
	return isPersonalModeOptionDisabled(request.Key)
}

// PersonalModeOption runs after RootAuth so unauthenticated requests cannot
// make the gateway parse option payloads before authorization.
func PersonalModeOption() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPut || !isPersonalModeOptionRequestDisabled(c) {
			c.Next()
			return
		}

		abortPersonalModeDisabled(c)
	}
}

func abortPersonalModeDisabled(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"code":    PersonalModeDisabledCode,
		"message": i18n.T(c, i18n.MsgPersonalModeDisabled),
	})
}
