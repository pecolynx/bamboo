package helper

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type LogConfig struct {
	Level string `yaml:"level"`
}

func InitLog(env string, cfg *LogConfig) error {
	formatter := &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyMsg: "message",
		},
	}
	logrus.SetFormatter(formatter)

	switch strings.ToLower(cfg.Level) {
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	default:
		logrus.Infof("Unsupported log level: %s", cfg.Level)
		logrus.SetLevel(logrus.WarnLevel)
	}

	logrus.SetOutput(os.Stdout)

	return nil
}
