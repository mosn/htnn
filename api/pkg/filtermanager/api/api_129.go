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

//go:build envoy1.29

package api

import (
	"sync/atomic"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

var (
	currLogLevel atomic.Int32
)

func GetLogLevel() LogType {
	lv := currLogLevel.Load()
	return LogType(lv)
}

func LogTrace(message string) {
	if GetLogLevel() > LogLevelTrace {
		return
	}
	api.LogTrace(message)
}

func LogDebug(message string) {
	if GetLogLevel() > LogLevelDebug {
		return
	}
	api.LogDebug(message)
}

func LogInfo(message string) {
	if GetLogLevel() > LogLevelInfo {
		return
	}
	api.LogInfo(message)
}

func LogWarn(message string) {
	if GetLogLevel() > LogLevelWarn {
		return
	}
	api.LogWarn(message)
}

func LogError(message string) {
	if GetLogLevel() > LogLevelError {
		return
	}
	api.LogError(message)
}

func LogCritical(message string) {
	if GetLogLevel() > LogLevelCritical {
		return
	}
	api.LogCritical(message)
}

func LogTracef(format string, v ...any) {
	if GetLogLevel() > LogLevelTrace {
		return
	}
	api.LogTracef(format, v...)
}

func LogDebugf(format string, v ...any) {
	if GetLogLevel() > LogLevelDebug {
		return
	}
	api.LogDebugf(format, v...)
}

func LogInfof(format string, v ...any) {
	if GetLogLevel() > LogLevelInfo {
		return
	}
	api.LogInfof(format, v...)
}

func LogWarnf(format string, v ...any) {
	if GetLogLevel() > LogLevelWarn {
		return
	}
	api.LogWarnf(format, v...)
}

func LogErrorf(format string, v ...any) {
	if GetLogLevel() > LogLevelError {
		return
	}
	api.LogErrorf(format, v...)
}

func LogCriticalf(format string, v ...any) {
	if GetLogLevel() > LogLevelCritical {
		return
	}
	api.LogCriticalf(format, v...)
}

type SecretManager interface {
}
