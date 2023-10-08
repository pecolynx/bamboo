package helper

import (
	"context"

	"github.com/pecolynx/bamboo"
)

func LogConfigFunc(ctx context.Context, headers map[string]string) context.Context {
	for k, v := range headers {
		if k == bamboo.RequestIDKey {
			ctx = bamboo.WithValue(ctx, bamboo.RequestIDContextKey, v)
		}
	}
	return ctx
}
