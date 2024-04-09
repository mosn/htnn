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

import "github.com/go-logr/logr"

var logger CtrlLogger = &logrWrapper{
	logger: logr.Logger{},
}

type CtrlLogger interface {
	Error(msg any)
	Errorf(format string, args ...any)
	Info(msg any)
	Infof(format string, args ...any)
}

func SetLogger(l CtrlLogger) {
	logger = l
}

// Error outputs a message at error level.
func Error(msg any) {
	logger.Error(msg)
}

// Errorf uses fmt.Sprintf to construct and log a message at error level.
func Errorf(format string, args ...any) {
	logger.Errorf(format, args...)
}

// Info outputs a message at info level.
func Info(msg any) {
	logger.Info(msg)
}

// Infof uses fmt.Sprintf to construct and log a message at info level.
func Infof(format string, args ...any) {
	logger.Infof(format, args...)
}
