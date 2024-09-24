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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogAPI(t *testing.T) {
	currLogLevel.Store(int32(LogLevelCritical))
	assert.Equal(t, LogLevelCritical, GetLogLevel())

	for _, s := range []struct {
		level string
		logf  func(string, ...any)
		log   func(string)
	}{
		{"Trace", LogTracef, LogTrace},
		{"Debug", LogDebugf, LogDebug},
		{"Info", LogInfof, LogInfo},
		{"Warn", LogWarnf, LogWarn},
		{"Error", LogErrorf, LogError},
	} {
		s.logf("test %s", s.level)
		s.log(s.level)
		// should not call api.LogXX directly - which will panic
	}
}
