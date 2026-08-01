package common

import (
	"context"

	"github.com/QuantumNous/new-api/constant"
)

type suppressRequestLogsContextKey struct{}

func WithSuppressedRequestLogs(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, suppressRequestLogsContextKey{}, true)
}

func RequestLogsSuppressed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if suppressed, _ := ctx.Value(suppressRequestLogsContextKey{}).(bool); suppressed {
		return true
	}
	suppressed, _ := ctx.Value(string(constant.ContextKeySuppressRequestLogs)).(bool)
	return suppressed
}
