//go:build ctrl

// the logger for control plane
package log

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	enc string
)

func init() {
	flag.CommandLine.StringVar(&enc, "log-encoder", "console", "Log encoding (one of 'json' or 'console', default to 'console')")

	// A minimal parser to work around flag package can be parsed only once.
	if len(os.Args) > 2 {
		for i, arg := range os.Args[1 : len(os.Args)-1] {
			if arg == "--log-encoder" {
				enc = os.Args[i+2]
			}
		}
	}

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
