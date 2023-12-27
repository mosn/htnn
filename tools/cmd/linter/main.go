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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// This tool does what the third party linters don't

func lintSite() error {
	// walk through directory
	return filepath.Walk(filepath.Join("site", "content"), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(path) == "content" {
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

func contains(set []string, s string) bool {
	targeted := false
	for _, e := range set {
		if s == e {
			targeted = true
			break
		}
	}
	return targeted

}

func lintFilename() error {
	excludedDir := []string{
		".git",
		".github",
		"site",
	}
	files, err := ioutil.ReadDir(".")
	if err != nil {
		return err
	}

	codeDir := []string{}
	for _, file := range files {
		if file.IsDir() && !contains(excludedDir, file.Name()) {
			codeDir = append(codeDir, file.Name())
		}
	}
	for _, dir := range codeDir {
		err := lintFilenameForCode(dir)
		if err != nil {
			return err
		}
	}
	docDir := []string{
		filepath.Join("site", "content"),
	}
	for _, dir := range docDir {
		err := lintFilenameForDoc(dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func lintFilenameForCode(root string) error {
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		base := filepath.Base(path)
		if base == "bin" {
			return filepath.SkipDir
		}

		ext := filepath.Ext(path)
		targeted := contains([]string{".go", ".proto", ".yaml", ".json", ".yml"}, ext)
		if !targeted {
			return nil
		}

		if base != "docker-compose.yml" && strings.ContainsRune(base, '-') {
			return fmt.Errorf("please use '_' instead of '-' in the code file name %s", path)
		}
		return nil
	})
}

func lintFilenameForDoc(root string) error {
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.ToLower(path) != path {
			return fmt.Errorf("name %s should be in lowercase", path)
		}

		base := filepath.Base(path)
		if info.IsDir() {
			if strings.ContainsRune(base, '_') {
				return fmt.Errorf("please use '-' instead of '_' in the doc directory name %s", path)
			}
		} else {
			// file required by the doc site framework
			if base == "search-index.md" {
				return nil
			}
			// other files, use the same rule as the code file
			if strings.ContainsRune(base, '-') {
				return fmt.Errorf("please use '_' instead of '-' in the code file name %s", path)
			}
		}
		return nil
	})
}

func main() {
	type linter func() error
	linters := []linter{
		lintSite,
		lintFilename,
	}
	for _, linter := range linters {
		err := linter()
		if err != nil {
			panic(err)
		}
	}
}
