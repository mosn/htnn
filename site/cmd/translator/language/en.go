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

package language

import (
	_ "embed"
)

var (
	//go:embed en.md
	enDemo string
)

func init() {
	registerLanguage("en", &en{})
}

type en struct {
}

func (l *en) Name() string {
	return "English"
}

func (l *en) Glossary() [][2]string {
	return nil
}

func (l *en) Rules() []string {
	return []string{}
}

func (l *en) Demo() string {
	return enDemo
}
