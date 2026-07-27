package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatusPersonalModeContract(t *testing.T) {
	originalOptionMap := common.OptionMap
	originalSelfUseMode := operation_setting.SelfUseModeEnabled
	originalEmailVerification := common.EmailVerificationEnabled
	originalGitHubOAuth := common.GitHubOAuthEnabled
	originalLinuxDOOAuth := common.LinuxDOOAuthEnabled
	originalTelegramOAuth := common.TelegramOAuthEnabled
	originalWeChatOAuth := common.WeChatAuthEnabled
	originalRegister := common.RegisterEnabled
	originalPasswordRegister := common.PasswordRegisterEnabled
	originalDiscord := *system_setting.GetDiscordSettings()
	originalOIDC := *system_setting.GetOIDCSettings()
	originalConsole := *console_setting.GetConsoleSetting()
	originalLegal := *system_setting.GetLegalSettings()
	originalCheckin := *operation_setting.GetCheckinSetting()
	t.Cleanup(func() {
		common.OptionMap = originalOptionMap
		operation_setting.SelfUseModeEnabled = originalSelfUseMode
		common.EmailVerificationEnabled = originalEmailVerification
		common.GitHubOAuthEnabled = originalGitHubOAuth
		common.LinuxDOOAuthEnabled = originalLinuxDOOAuth
		common.TelegramOAuthEnabled = originalTelegramOAuth
		common.WeChatAuthEnabled = originalWeChatOAuth
		common.RegisterEnabled = originalRegister
		common.PasswordRegisterEnabled = originalPasswordRegister
		*system_setting.GetDiscordSettings() = originalDiscord
		*system_setting.GetOIDCSettings() = originalOIDC
		*console_setting.GetConsoleSetting() = originalConsole
		*system_setting.GetLegalSettings() = originalLegal
		*operation_setting.GetCheckinSetting() = originalCheckin
	})

	common.OptionMap = map[string]string{}
	common.EmailVerificationEnabled = true
	common.GitHubOAuthEnabled = true
	common.LinuxDOOAuthEnabled = true
	common.TelegramOAuthEnabled = true
	common.WeChatAuthEnabled = true
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	system_setting.GetDiscordSettings().Enabled = true
	system_setting.GetOIDCSettings().Enabled = true
	console := console_setting.GetConsoleSetting()
	console.ApiInfoEnabled = true
	console.ApiInfo = `[{"url":"https://example.com","route":"api","description":"api","color":"blue"}]`
	console.AnnouncementsEnabled = true
	console.Announcements = `[{"content":"notice","publishDate":"2026-07-27T00:00:00Z"}]`
	console.FAQEnabled = true
	console.FAQ = `[{"question":"q","answer":"a"}]`
	system_setting.GetLegalSettings().UserAgreement = "agreement"
	system_setting.GetLegalSettings().PrivacyPolicy = "privacy"
	operation_setting.GetCheckinSetting().Enabled = true

	requestStatus := func() map[string]any {
		response := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(response)
		context.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
		GetStatus(context)

		var payload struct {
			Success bool           `json:"success"`
			Message string         `json:"message"`
			Data    map[string]any `json:"data"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
		assert.Equal(t, http.StatusOK, response.Code)
		assert.True(t, payload.Success)
		assert.Empty(t, payload.Message)
		return payload.Data
	}

	operation_setting.SelfUseModeEnabled = false
	standard := requestStatus()
	assert.Equal(t, true, standard["register_enabled"])
	assert.Equal(t, true, standard["github_oauth"])
	assert.Contains(t, standard, "api_info")
	assert.Contains(t, standard, "announcements")
	assert.Contains(t, standard, "faq")

	operation_setting.SelfUseModeEnabled = true
	personal := requestStatus()
	for _, key := range []string{
		"register_enabled",
		"password_register_enabled",
		"email_verification",
		"github_oauth",
		"discord_oauth",
		"linuxdo_oauth",
		"telegram_oauth",
		"wechat_login",
		"oidc_enabled",
		"oauth_register_enabled",
		"api_info_enabled",
		"announcements_enabled",
		"faq_enabled",
		"user_agreement_enabled",
		"privacy_policy_enabled",
		"checkin_enabled",
	} {
		assert.Equal(t, false, personal[key], key)
	}
	assert.Equal(t, true, personal["self_use_mode_enabled"])
	assert.NotContains(t, personal, "api_info")
	assert.NotContains(t, personal, "announcements")
	assert.NotContains(t, personal, "faq")
	assert.NotContains(t, personal, "custom_oauth_providers")
}
