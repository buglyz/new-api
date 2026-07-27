package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersonalModeCapabilityMatrix(t *testing.T) {
	t.Run("disabled capabilities use exact method and registered path", func(t *testing.T) {
		disabled := []personalModeRoute{
			{http.MethodPost, "/api/user/register"},
			{http.MethodGet, "/api/oauth/:provider"},
			{http.MethodDelete, "/api/user/self"},
			{http.MethodPost, "/api/user/waffo-pancake/pay"},
			{http.MethodDelete, "/api/user/:id"},
			{http.MethodDelete, "/api/user/:id/reset_passkey"},
			{http.MethodGet, "/api/user/2fa/stats"},
			{http.MethodDelete, "/api/user/:id/2fa"},
			{http.MethodPatch, "/api/subscription/admin/plans/:id"},
			{http.MethodDelete, "/api/redemption/:id"},
			{http.MethodPost, "/api/option/payment_compliance"},
			{http.MethodPut, "/api/custom-oauth-provider/:id"},
		}
		for _, route := range disabled {
			assert.True(t, isPersonalModeRouteDisabled(route.method, route.fullPath), route)
		}

		assert.False(t, isPersonalModeRouteDisabled(http.MethodPost, "/api/pricing"))
		assert.False(t, isPersonalModeRouteDisabled(http.MethodGet, "/api/oauth/github"))
		assert.False(t, isPersonalModeRouteDisabled(http.MethodGet, "/api/user/self"))
	})

	t.Run("operational allowlist remains available", func(t *testing.T) {
		allowed := []personalModeRoute{
			{http.MethodPost, "/api/user/login"},
			{http.MethodPost, "/api/user/auth/refresh"},
			{http.MethodPost, "/api/user/auth/logout"},
			{http.MethodGet, "/api/user/self"},
			{http.MethodPut, "/api/user/self"},
			{http.MethodGet, "/api/user/sessions"},
			{http.MethodDelete, "/api/user/sessions/:sid"},
			{http.MethodGet, "/api/user/2fa/status"},
			{http.MethodPost, "/api/user/passkey/register/begin"},
			{http.MethodGet, "/api/channel/:id"},
			{http.MethodPost, "/api/channel/:id/codex/refresh"},
			{http.MethodGet, "/api/models/:id"},
			{http.MethodGet, "/api/group/"},
			{http.MethodPost, "/api/token/"},
			{http.MethodGet, "/api/log/self"},
			{http.MethodGet, "/api/data/self"},
			{http.MethodGet, "/api/system-info/instances"},
			{http.MethodGet, "/api/performance/stats"},
			{http.MethodGet, "/api/perf-metrics"},
			{http.MethodGet, "/api/perf-metrics/summary"},
			{http.MethodGet, "/api/ratio_config"},
			{http.MethodPost, "/api/ratio_sync/fetch"},
			{http.MethodGet, "/api/setup"},
			{http.MethodGet, "/api/status"},
			{http.MethodPost, "/v1/chat/completions"},
		}
		for _, route := range allowed {
			assert.False(t, isPersonalModeRouteDisabled(route.method, route.fullPath), route)
		}
	})
}

func TestPersonalModeMiddleware(t *testing.T) {
	require.NoError(t, i18n.Init())
	original := operation_setting.SelfUseModeEnabled
	t.Cleanup(func() { operation_setting.SelfUseModeEnabled = original })

	newRouter := func() *gin.Engine {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(PersonalMode())
		router.DELETE("/api/user/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})
		router.POST("/api/channel/:id/codex/refresh", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})
		router.GET("/api/operational", PersonalModeAdmin(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})
		router.PUT("/api/option/", PersonalModeOption(), func(c *gin.Context) {
			var request struct {
				Key string `json:"key"`
			}
			require.NoError(t, common.DecodeJson(c.Request.Body, &request))
			c.JSON(http.StatusOK, gin.H{"success": true, "key": request.Key})
		})
		return router
	}

	t.Run("mode off preserves handler behavior", func(t *testing.T) {
		operation_setting.SelfUseModeEnabled = false
		response := httptest.NewRecorder()
		newRouter().ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/user/42", nil))

		assert.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{"success":true}`, response.Body.String())
	})

	t.Run("mode on blocks a dynamic disabled path", func(t *testing.T) {
		operation_setting.SelfUseModeEnabled = true
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodDelete, "/api/user/42", nil)
		request.Header.Set("Accept-Language", "zh-CN")
		newRouter().ServeHTTP(response, request)

		assert.Equal(t, http.StatusForbidden, response.Code)
		var payload struct {
			Success bool   `json:"success"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
		assert.False(t, payload.Success)
		assert.Equal(t, PersonalModeDisabledCode, payload.Code)
		assert.Equal(t, "个人模式下此功能不可用", payload.Message)
	})

	t.Run("mode on allows an operational dynamic path", func(t *testing.T) {
		operation_setting.SelfUseModeEnabled = true
		response := httptest.NewRecorder()
		newRouter().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/channel/42/codex/refresh", nil))

		assert.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{"success":true}`, response.Body.String())
	})

	t.Run("mode on requires admin authentication for operational data", func(t *testing.T) {
		operation_setting.SelfUseModeEnabled = true
		response := httptest.NewRecorder()
		newRouter().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/operational", nil))

		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("mode on blocks commercial option keys without blocking other settings", func(t *testing.T) {
		operation_setting.SelfUseModeEnabled = true
		for _, key := range []string{"WaffoEnabled", "StripeApiSecret", "RegisterEnabled", "oidc.enabled"} {
			blocked := httptest.NewRecorder()
			blockedRequest := httptest.NewRequest(
				http.MethodPut,
				"/api/option/",
				strings.NewReader(`{"key":"`+key+`","value":true}`),
			)
			blockedRequest.Header.Set("Content-Type", "application/json")
			newRouter().ServeHTTP(blocked, blockedRequest)
			assert.Equal(t, http.StatusForbidden, blocked.Code, key)
		}

		allowed := httptest.NewRecorder()
		allowedRequest := httptest.NewRequest(
			http.MethodPut,
			"/api/option/",
			strings.NewReader(`{"key":"RetryTimes","value":3}`),
		)
		allowedRequest.Header.Set("Content-Type", "application/json")
		newRouter().ServeHTTP(allowed, allowedRequest)
		assert.Equal(t, http.StatusOK, allowed.Code)
		assert.JSONEq(t, `{"success":true,"key":"RetryTimes"}`, allowed.Body.String())
	})
}
