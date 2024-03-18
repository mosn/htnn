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

var logger logr.Logger

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

	logger = zapr.NewLoggerWithOptions(zapLog)
}

func Logger() logr.Logger {
	return logger
}

// Error outputs a message at error level.
func Error(msg any) {
	logger.Error(nil, fmt.Sprint(msg))
}

// Errorf uses fmt.Sprintf to construct and log a message at error level.
func Errorf(format string, args ...any) {
	logger.Error(nil, fmt.Sprintf(format, args...))
}

// Info outputs a message at info level.
func Info(msg any) {
	logger.Info(fmt.Sprint(msg))
}

// Infof uses fmt.Sprintf to construct and log a message at info level.
func Infof(format string, args ...any) {
	logger.Info(fmt.Sprintf(format, args...))
}
