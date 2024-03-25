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
