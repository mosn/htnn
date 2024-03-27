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

//go:build so

// the logger for data plane
package log

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"

	"mosn.io/htnn/api/pkg/filtermanager/api"
)

func ptrstr(s string) *string {
	return &s
}

func init() {
	// Name of this file guarantees that SetLogger runs after DefaultLogger init.
	SetLogger(DefaultLogger.WithSink(&EnvoyLogSink{
		Formatter: funcr.NewFormatter(funcr.Options{
			LogInfoLevel: ptrstr(""),
		}),
	}))
}

type EnvoyLogSink struct {
	funcr.Formatter
}

func (l *EnvoyLogSink) Init(info logr.RuntimeInfo) {
}

func (l *EnvoyLogSink) Enabled(level int) bool {
	// We don't use V-level log
	return true
}

func (l *EnvoyLogSink) Info(level int, msg string, keysAndValues ...any) {
	prefix, s := l.Formatter.FormatInfo(level, msg, keysAndValues)
	if prefix != "" {
		api.LogInfof("[%s] %s", prefix, s)
	} else {
		api.LogInfo(s)
	}
}

func (l *EnvoyLogSink) Error(err error, msg string, keysAndValues ...any) {
	prefix, s := l.Formatter.FormatError(err, msg, keysAndValues)
	if prefix != "" {
		api.LogErrorf("[%s] %s", prefix, s)
	} else {
		api.LogError(s)
	}
}

func (l *EnvoyLogSink) WithValues(keysAndValues ...any) logr.LogSink {
	nl := &EnvoyLogSink{
		Formatter: l.Formatter, // copy of Formatter
	}
	nl.Formatter.AddValues(keysAndValues)
	return nl
}

func (l *EnvoyLogSink) WithName(name string) logr.LogSink {
	nl := &EnvoyLogSink{
		Formatter: l.Formatter,
	}
	nl.Formatter.AddName(name)
	return nl
}
