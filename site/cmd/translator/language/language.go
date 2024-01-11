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

type Language interface {
	Name() string
	Glossary() string
	Rules() []string
	Demo() string
}

var (
	languages = map[string]Language{}
)

func registerLanguage(code string, la Language) {
	languages[code] = la
}

func LookupLanguage(code string) Language {
	return languages[code]
}
