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
	"mosn.io/htnn/controller/internal/log"
)

type logger struct {
	name string
}

type RegistryLogger interface {
	Errorf(format string, args ...any)
	Infof(format string, args ...any)
}

type RegistryLoggerOptions struct {
	Name string
}

func NewLogger(opts *RegistryLoggerOptions) RegistryLogger {
	if opts == nil {
		panic("logger options are required")
	}
	return &logger{
		name: opts.Name,
	}
}

const (
	fmtTail = ", registry: %s"
)

func (l *logger) Errorf(format string, args ...any) {
	log.Errorf(format+fmtTail, append(args, l.name)...)
}

func (l *logger) Infof(format string, args ...any) {
	log.Infof(format+fmtTail, append(args, l.name)...)
}
