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
