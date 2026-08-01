package logger

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLogErrorSuppressesMarkedRequestContext(t *testing.T) {
	var output bytes.Buffer
	common.LogWriterMu.Lock()
	previousWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &output
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = previousWriter
		common.LogWriterMu.Unlock()
	})

	LogError(common.WithSuppressedRequestLogs(context.Background()), "upstream-secret")
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ginContext, constant.ContextKeySuppressRequestLogs, true)
	LogError(ginContext, "gin-upstream-secret")

	assert.NotContains(t, output.String(), "upstream-secret")
	assert.NotContains(t, output.String(), "gin-upstream-secret")
}
