package helper

import (
	"context"

	"github.com/pecolynx/bamboo/internal"
)

func LogConfigFunc(ctx context.Context, headers map[string]string) context.Context {
	for k, v := range headers {
		ctx = internal.With(ctx, internal.Str(k, v))
	}
	return ctx
}
