//go:build so

// the logger for data plane
package log

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"

	"mosn.io/moe/pkg/filtermanager/api"
)

func init() {
	// Name of this file guarantees that SetLogger runs after DefaultLogger init.
	SetLogger(DefaultLogger.WithSink(&EnvoyLogSink{
		Formatter: funcr.NewFormatter(funcr.Options{}),
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
