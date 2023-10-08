package helper

import (
	"context"

	"github.com/pecolynx/bamboo/sloghelper"
)

func LogConfigFunc(ctx context.Context, headers map[string]string) context.Context {
	for k, v := range headers {
		if k == sloghelper.RequestIDKey {
			ctx = sloghelper.WithValue(ctx, sloghelper.RequestIDContextKey, v)
		}
	}
	return ctx
}
