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

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// This tool does what the third party linters don't

func lint_site() error {
	// walk through directory
	err := os.Chdir("./site")
	if err != nil {
		return err
	}

	return filepath.Walk("content", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.ToLower(path) != path {
			return fmt.Errorf("name %s should be in lowercase", path)
		}
		if info.IsDir() {
			if path == "content" {
				return nil
			}
			// check index
			if _, err := os.Stat(filepath.Join(path, "_index.md")); err != nil {
				_, err = os.Stat(filepath.Join(path, "_index.html"))
				if err != nil {
					return err
				}
				return err
			}
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".md" {
			return fmt.Errorf("file %s has unexpected extension", path)
		}

		if ext == ".md" {
			if !bytes.HasPrefix(data, []byte("---")) {
				return fmt.Errorf("header is missing in %s", path)
			}

			base := filepath.Base(path)
			if base == "search-index.md" {
				return nil
			}

			scanner := bufio.NewScanner(bytes.NewReader(data))
			scanner.Scan()
			title := false
			for scanner.Scan() {
				text := scanner.Text()
				if text == "---" {
					break
				}
				if strings.HasPrefix(text, "title: ") {
					title = true
				}
			}

			if !title {
				return fmt.Errorf("title is missing in %s", path)
			}
		}
		return nil
	})
}

func main() {
	err := lint_site()
	if err != nil {
		panic(err)
	}
}
