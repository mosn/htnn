//go:build ctrl

// the logger for control plane
package log

import (
	"flag"
	"fmt"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	enc string
)

func bindFlags(fs *flag.FlagSet) {
	// TODO: unitfy flag style
	fs.StringVar(&enc, "log-encoder", "console", "Log encoding (one of 'json' or 'console', default to 'console')")
}

func init() {
	bindFlags(flag.CommandLine)
	flag.Parse()

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Encoding = enc
	zapLog, err := cfg.Build(
		zap.WithCaller(false), // enable caller will make log nearly 10 times slow
	)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}

	// Name of this file guarantees that SetLogger runs after DefaultLogger init.
	SetLogger(zapr.NewLoggerWithOptions(zapLog))
}
