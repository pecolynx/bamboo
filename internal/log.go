// The MIT License (MIT)

// Copyright (c) 2016-2020 Containous SAS; 2020-2021 Traefik Labs

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// https://github.com/traefik/traefik

package internal

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type contextKey int

const (
	loggerKey contextKey = iota
)

// Logger the Traefik logger
type Logger interface {
	logrus.FieldLogger
	WriterLevel(logrus.Level) *io.PipeWriter
}

var (
	mainLogger Logger
)

func init() {
	mainLogger = logrus.StandardLogger()
	logrus.SetOutput(os.Stdout)
}

// // SetLogger sets the logger.
// func SetLogger(l Logger) {
// 	mainLogger = l
// }

// // SetOutput sets the standard logger output.
// func SetOutput(out io.Writer) {
// 	logrus.SetOutput(out)
// }

// // SetFormatter sets the standard logger formatter.
// func SetFormatter(formatter logrus.Formatter) {
// 	logrus.SetFormatter(formatter)
// }

// // SetLevel sets the standard logger level.
// func SetLevel(level logrus.Level) {
// 	logrus.SetLevel(level)
// }

// // GetLevel returns the standard logger level.
// func GetLevel() logrus.Level {
// 	return logrus.GetLevel()
// }

// Str adds a string field
func Str(key, value string) func(logrus.Fields) {
	return func(fields logrus.Fields) {
		fields[key] = value
	}
}

// With Adds fields
func With(ctx context.Context, opts ...func(logrus.Fields)) context.Context {
	logger := FromContext(ctx)

	fields := make(logrus.Fields)
	for _, opt := range opts {
		opt(fields)
	}
	logger = logger.WithFields(fields)

	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext Gets the logger from context
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		panic("nil context")
	}

	logger, ok := ctx.Value(loggerKey).(Logger)
	if !ok {
		logger = mainLogger
	}

	return logger
}
