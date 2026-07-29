package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRetiredFrontendAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	_, hasAsyncCleanup := routes[http.MethodPost+" /api/system-task/log-cleanup"]
	_, hasDirectDelete := routes[http.MethodDelete+" /api/log/"]
	_, hasConsoleMigration := routes[http.MethodPost+" /api/option/migrate_console_setting"]
	assert.True(t, hasAsyncCleanup)
	assert.False(t, hasDirectDelete)
	assert.False(t, hasConsoleMigration)
}

func TestSelfUseOnlyAPIRouteContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	retiredRoutes := []string{
		http.MethodPost + " /api/user/register",
		http.MethodGet + " /api/user/groups",
		http.MethodGet + " /api/oauth/:provider",
		http.MethodGet + " /api/rankings",
		http.MethodGet + " /api/user/aff",
		http.MethodPost + " /api/user/topup",
		http.MethodGet + " /api/subscription/plans",
		http.MethodGet + " /api/redemption/",
		http.MethodGet + " /api/user/",
	}
	for _, route := range retiredRoutes {
		assert.NotContains(t, routes, route)
	}

	retainedRoutes := []string{
		http.MethodPost + " /api/user/login",
		http.MethodPost + " /api/user/passkey/login/begin",
		http.MethodGet + " /api/user/self",
		http.MethodGet + " /api/user/passkey",
		http.MethodGet + " /api/user/2fa/status",
		http.MethodGet + " /api/channel/",
		http.MethodGet + " /api/channel/reliability",
		http.MethodGet + " /api/token/",
		http.MethodGet + " /api/pricing",
		http.MethodGet + " /api/log/",
	}
	for _, route := range retainedRoutes {
		assert.Contains(t, routes, route)
	}
}
