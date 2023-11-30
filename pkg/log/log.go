package log

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

var DefaultLogger = initDefaultLogger()

func initDefaultLogger() logr.Logger {
	var log logr.Logger

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	log = zapr.NewLogger(zapLog)
	return log
}

func SetLogger(logger logr.Logger) {
	DefaultLogger = logger
}
