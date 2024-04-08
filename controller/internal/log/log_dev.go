// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var devLogger logr.Logger

func InitLogger(enc string) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Encoding = enc
	zapLog, err := cfg.Build(
		zap.WithCaller(false), // enable caller will make log nearly 10 times slow
	)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}

	devLogger = zapr.NewLoggerWithOptions(zapLog)

	SetLogger(wrapLogr(devLogger))
}

func Logger() logr.Logger {
	return devLogger
}

type logrWrapper struct {
	logger logr.Logger
}

func wrapLogr(l logr.Logger) CtrlLogger {
	return &logrWrapper{
		logger: l,
	}
}

func (l *logrWrapper) Error(msg any) {
	l.logger.Error(nil, fmt.Sprint(msg))
}

func (l *logrWrapper) Errorf(format string, args ...any) {
	l.logger.Error(nil, fmt.Sprintf(format, args...))
}

func (l *logrWrapper) Info(msg any) {
	l.logger.Info(fmt.Sprint(msg))
}

func (l *logrWrapper) Infof(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}
