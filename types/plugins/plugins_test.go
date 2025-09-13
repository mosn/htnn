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

package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"mosn.io/htnn/api/pkg/plugins"
)

func snakeToCamel(s string) string {
	words := strings.Split(s, "_")
	for i := 1; i < len(words); i++ {
		words[i] = cases.Title(language.Und, cases.NoLower).String(words[i])
	}
	return strings.Join(words, "")
}

func getSecondColumn(line string) string {
	cols := strings.Split(line, "|")
	if len(cols) < 3 {
		return ""
	}
	return strings.TrimSpace(cols[2])
}

func TestCheckPluginAttributes(t *testing.T) {
	err := filepath.Walk(filepath.Join("..", "..", "site", "content", "en", "docs", "reference", "plugins"), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "_index.md" {
			return nil
		}

		var plugin string
		filename := filepath.Base(path)[:len(filepath.Base(path))-3]
		if filename == "network_rbac" {
			plugin = "networkRBAC"
		} else if filename == "ai_content_security" {
			plugin = "AIContentSecurity"
		} else if filename == "limit_token" {
			plugin = "limitToken"
		} else {
			plugin = snakeToCamel(filename)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		p := plugins.LoadPluginType(plugin)
		if p == nil {
			return fmt.Errorf("plugin %s not found", plugin)
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Scan()
		for scanner.Scan() {
			text := scanner.Text()
			if text == "## Attribute" {
				for i := 0; i < 3; i++ {
					scanner.Scan() // drop table title
				}

				scanner.Scan()
				ty := getSecondColumn(scanner.Text())
				scanner.Scan()
				order := getSecondColumn(scanner.Text())

				assert.Equal(t, p.Type().String(), ty, "plugin %s type mismatch", plugin)
				assert.Equal(t, p.Order().Position.String(), order, "plugin %s order mismatch", plugin)
				break
			}
		}
		return nil
	})
	assert.Nil(t, err)
}
