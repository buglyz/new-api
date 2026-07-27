package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const PersonalModeDisabledCode = "PERSONAL_MODE_DISABLED"

type personalModeRoute struct {
	method   string
	fullPath string
}

var personalModeDisabledRoutes = map[personalModeRoute]struct{}{
	// Public account acquisition and recovery.
	{http.MethodGet, "/api/verification"}:                    {},
	{http.MethodGet, "/api/reset_password"}:                  {},
	{http.MethodPost, "/api/user/reset"}:                     {},
	{http.MethodPost, "/api/user/register"}:                  {},
	{http.MethodPost, "/api/oauth/state"}:                    {},
	{http.MethodPost, "/api/oauth/email/bind"}:               {},
	{http.MethodGet, "/api/oauth/wechat"}:                    {},
	{http.MethodPost, "/api/oauth/wechat/bind"}:              {},
	{http.MethodGet, "/api/oauth/telegram/login"}:            {},
	{http.MethodPost, "/api/oauth/telegram/bind/start"}:      {},
	{http.MethodGet, "/api/oauth/telegram/bind/:flow_token"}: {},
	{http.MethodGet, "/api/oauth/:provider"}:                 {},

	// Public marketing and legal content.
	{http.MethodGet, "/api/notice"}:            {},
	{http.MethodGet, "/api/user-agreement"}:    {},
	{http.MethodGet, "/api/privacy-policy"}:    {},
	{http.MethodGet, "/api/about"}:             {},
	{http.MethodGet, "/api/home_page_content"}: {},
	{http.MethodGet, "/api/pricing"}:           {},
	{http.MethodGet, "/api/rankings"}:          {},

	// Payment callbacks and webhooks.
	{http.MethodPost, "/api/stripe/webhook"}:             {},
	{http.MethodPost, "/api/creem/webhook"}:              {},
	{http.MethodPost, "/api/waffo/webhook"}:              {},
	{http.MethodPost, "/api/waffo-pancake/webhook/:env"}: {},
	{http.MethodGet, "/api/user/epay/notify"}:            {},
	{http.MethodPost, "/api/user/epay/notify"}:           {},
	{http.MethodGet, "/api/subscription/epay/notify"}:    {},
	{http.MethodPost, "/api/subscription/epay/notify"}:   {},
	{http.MethodGet, "/api/subscription/epay/return"}:    {},
	{http.MethodPost, "/api/subscription/epay/return"}:   {},

	// Wallet, check-in, affiliate, and OAuth binding operations.
	{http.MethodDelete, "/api/user/self"}:                            {},
	{http.MethodGet, "/api/user/aff"}:                                {},
	{http.MethodGet, "/api/user/topup/info"}:                         {},
	{http.MethodGet, "/api/user/topup/self"}:                         {},
	{http.MethodPost, "/api/user/topup"}:                             {},
	{http.MethodPost, "/api/user/pay"}:                               {},
	{http.MethodPost, "/api/user/amount"}:                            {},
	{http.MethodPost, "/api/user/stripe/pay"}:                        {},
	{http.MethodPost, "/api/user/stripe/amount"}:                     {},
	{http.MethodPost, "/api/user/creem/pay"}:                         {},
	{http.MethodPost, "/api/user/waffo/amount"}:                      {},
	{http.MethodPost, "/api/user/waffo/pay"}:                         {},
	{http.MethodPost, "/api/user/waffo-pancake/amount"}:              {},
	{http.MethodPost, "/api/user/waffo-pancake/pay"}:                 {},
	{http.MethodPost, "/api/user/aff_transfer"}:                      {},
	{http.MethodGet, "/api/user/checkin"}:                            {},
	{http.MethodPost, "/api/user/checkin"}:                           {},
	{http.MethodGet, "/api/user/oauth/bindings"}:                     {},
	{http.MethodDelete, "/api/user/oauth/bindings/:provider_id"}:     {},
	{http.MethodGet, "/api/user/:id/oauth/bindings"}:                 {},
	{http.MethodDelete, "/api/user/:id/oauth/bindings/:provider_id"}: {},
	{http.MethodDelete, "/api/user/:id/bindings/:binding_type"}:      {},

	// Administrative user CRUD and top-up fulfillment.
	{http.MethodGet, "/api/user/"}:                     {},
	{http.MethodGet, "/api/user/search"}:               {},
	{http.MethodGet, "/api/user/:id"}:                  {},
	{http.MethodPost, "/api/user/"}:                    {},
	{http.MethodPost, "/api/user/manage"}:              {},
	{http.MethodPut, "/api/user/"}:                     {},
	{http.MethodDelete, "/api/user/:id"}:               {},
	{http.MethodDelete, "/api/user/:id/reset_passkey"}: {},
	{http.MethodGet, "/api/user/2fa/stats"}:            {},
	{http.MethodDelete, "/api/user/:id/2fa"}:           {},
	{http.MethodGet, "/api/user/topup"}:                {},
	{http.MethodPost, "/api/user/topup/complete"}:      {},

	// Subscription and redemption code management.
	{http.MethodGet, "/api/subscription/plans"}:                                    {},
	{http.MethodGet, "/api/subscription/self"}:                                     {},
	{http.MethodPut, "/api/subscription/self/preference"}:                          {},
	{http.MethodPost, "/api/subscription/balance/pay"}:                             {},
	{http.MethodPost, "/api/subscription/epay/pay"}:                                {},
	{http.MethodPost, "/api/subscription/stripe/pay"}:                              {},
	{http.MethodPost, "/api/subscription/creem/pay"}:                               {},
	{http.MethodPost, "/api/subscription/waffo-pancake/pay"}:                       {},
	{http.MethodGet, "/api/subscription/admin/plans"}:                              {},
	{http.MethodPost, "/api/subscription/admin/plans"}:                             {},
	{http.MethodPut, "/api/subscription/admin/plans/:id"}:                          {},
	{http.MethodPatch, "/api/subscription/admin/plans/:id"}:                        {},
	{http.MethodPost, "/api/subscription/admin/bind"}:                              {},
	{http.MethodPost, "/api/subscription/admin/plans/:id/subscriptions/reset"}:     {},
	{http.MethodGet, "/api/subscription/admin/users/:id/subscriptions"}:            {},
	{http.MethodPost, "/api/subscription/admin/users/:id/subscriptions"}:           {},
	{http.MethodPost, "/api/subscription/admin/users/:id/subscriptions/reset"}:     {},
	{http.MethodPost, "/api/subscription/admin/user_subscriptions/:id/invalidate"}: {},
	{http.MethodDelete, "/api/subscription/admin/user_subscriptions/:id"}:          {},
	{http.MethodGet, "/api/redemption/"}:                                           {},
	{http.MethodGet, "/api/redemption/search"}:                                     {},
	{http.MethodGet, "/api/redemption/:id"}:                                        {},
	{http.MethodPost, "/api/redemption/"}:                                          {},
	{http.MethodPut, "/api/redemption/"}:                                           {},
	{http.MethodDelete, "/api/redemption/invalid"}:                                 {},
	{http.MethodDelete, "/api/redemption/:id"}:                                     {},

	// Commercial configuration with dedicated endpoints.
	{http.MethodPost, "/api/option/payment_compliance"}:                        {},
	{http.MethodGet, "/api/option/waffo-pancake/catalog"}:                      {},
	{http.MethodPost, "/api/option/waffo-pancake/pair"}:                        {},
	{http.MethodPost, "/api/option/waffo-pancake/save"}:                        {},
	{http.MethodPost, "/api/option/waffo-pancake/subscription-product"}:        {},
	{http.MethodGet, "/api/option/waffo-pancake/subscription-product-options"}: {},

	// Custom OAuth provider administration.
	{http.MethodPost, "/api/custom-oauth-provider/discovery"}: {},
	{http.MethodGet, "/api/custom-oauth-provider/"}:           {},
	{http.MethodGet, "/api/custom-oauth-provider/:id"}:        {},
	{http.MethodPost, "/api/custom-oauth-provider/"}:          {},
	{http.MethodPut, "/api/custom-oauth-provider/:id"}:        {},
	{http.MethodDelete, "/api/custom-oauth-provider/:id"}:     {},
}

var personalModeDisabledOptionKeys = map[string]struct{}{
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

func isPersonalModeRouteDisabled(method, fullPath string) bool {
	_, disabled := personalModeDisabledRoutes[personalModeRoute{method: method, fullPath: fullPath}]
	return disabled
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

func PersonalMode() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !operation_setting.SelfUseModeEnabled || !isPersonalModeRouteDisabled(c.Request.Method, c.FullPath()) {
			c.Next()
			return
		}

		abortPersonalModeDisabled(c)
	}
}

// PersonalModeAdmin keeps operational endpoints available to administrators
// without leaving them public when the console runs in personal mode.
func PersonalModeAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !operation_setting.SelfUseModeEnabled {
			c.Next()
			return
		}

		AdminAuth()(c)
	}
}

// PersonalModeOnly limits fork-specific operational APIs to personal mode.
// Standard mode keeps its original surface and receives the same stable 403
// contract used by the personal-mode capability matrix.
func PersonalModeOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if operation_setting.SelfUseModeEnabled {
			c.Next()
			return
		}
		abortPersonalModeDisabled(c)
	}
}

// PersonalModeOption runs after RootAuth so unauthenticated requests cannot
// make the gateway parse option payloads before authorization.
func PersonalModeOption() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !operation_setting.SelfUseModeEnabled || c.Request.Method != http.MethodPut || !isPersonalModeOptionRequestDisabled(c) {
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
