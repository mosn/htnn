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
	//go:embed zh-Hans.md
	zhHansDemo string
)

func init() {
	registerLanguage("zh-Hans", &zhHans{})
}

type zhHans struct {
}

func (l *zhHans) Name() string {
	return "Simplified Chinese"
}

func (l *zhHans) Glossary() string {
	return `
## Description -> ## 说明
## Usage -> ## 用法
`
}

func (l *zhHans) Rules() []string {
	return []string{
		`put a space between the Chinese content and the English words or numbers`,
		`True in the Required column of the Markdown form should be translated as "是" and False as "否"`,
		`markdown table headers should be translated strictly according to the following mapping:
  a. "| Name | Type | Required | Validation | Description" translates to "| 名称  | 类型 | 必选 | 校验规则| 说明"`,
	}
}

func (l *zhHans) Demo() string {
	return zhHansDemo
}
