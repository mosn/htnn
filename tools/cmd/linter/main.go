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
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	protoparser "github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/parser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"mosn.io/htnn/api/pkg/plugins"
	_ "mosn.io/htnn/types/plugins"
)

// This tool does what the third party linters don't

const (
	LangEn     = "en"
	LangZhHans = "zh-hans"
)

func lintSite() error {
	enDocs := map[string]struct{}{}
	zhHansDocs := map[string]struct{}{}

	// walk through directory
	err := filepath.Walk(filepath.Join("site", "content"), func(path string, info fs.FileInfo, err error) error {
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
					return fmt.Errorf("directory %s is missing _index.md or _index.html as the _index file: %s", path, err)
				}
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".html" && ext != ".md" {
			return fmt.Errorf("file %s has unexpected extension", path)
		}

		if ext == ".md" {
			if strings.Contains(path, "en/") {
				enDocs[strings.TrimPrefix(path, "site/content/en/")] = struct{}{}
			} else if strings.Contains(path, "zh-hans/") {
				zhHansDocs[strings.TrimPrefix(path, "site/content/zh-hans/")] = struct{}{}
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

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
	if err != nil {
		return err
	}

	// don't treat this as an error
	for doc := range enDocs {
		if _, ok := zhHansDocs[doc]; !ok {
			fmt.Printf("file %s is missing in Simplified Chinese documentation\n", doc)
		}
	}
	for doc := range zhHansDocs {
		zhMs, err := readDoc(filepath.Join("site", "content", "zh-hans", doc), LangZhHans)
		if err != nil {
			return err
		}

		if _, ok := enDocs[doc]; !ok {
			fmt.Printf("file %s is missing in English documentation\n", doc)
		} else {
			enMs, err := readDoc(filepath.Join("site", "content", "en", doc), LangEn)
			if err != nil {
				return err
			}

			if !reflect.DeepEqual(enMs, zhMs) {
				zhMsOut, _ := json.MarshalIndent(zhMs, "", "  ")
				enMsOut, _ := json.MarshalIndent(enMs, "", "  ")
				fmt.Printf("mismatched fields in %s:\nSimpilified Chinese %s\nEnglish %s\n", doc, zhMsOut, enMsOut)
			}
		}
	}

	return nil
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
		"external",
		"manifests",
	}
	files, err := os.ReadDir(".")
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

func readPackageName(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			return line, nil
		}
	}
	return "", nil
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

		if strings.ToLower(base) != base {
			return fmt.Errorf("name %s should be in lowercase", path)
		}

		if ext == ".go" {
			line, err := readPackageName(path)
			if err != nil {
				return err
			}

			if line != "package main" {
				if line != "package "+filepath.Base(filepath.Dir(path)) {
					return fmt.Errorf("package name should be the same as the directory name in %s", path)
				}
			}
		}

		return nil
	})
}

// snakeToCamel converts a snake_case string to a camelCase string.
func snakeToCamel(s string) string {
	words := strings.Split(s, "_")
	for i := 1; i < len(words); i++ {
		words[i] = cases.Title(language.Und, cases.NoLower).String(words[i])
	}
	return strings.Join(words, "")
}

func filenameToPluginName(name string) string {
	if name == "network_rbac" {
		return "networkRBAC"
	}
	if name == "ai_content_security" {
		return "AIContentSecurity"
	}
	if name == "limit_token" {
		return "limitToken"
	}
	return snakeToCamel(name)
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

var (
	Categories = []string{"plugins", "registries"}
)

func lintConfiguration() error {
	for _, t := range Categories {
		err := lintConfigurationByCategory(t)
		if err != nil {
			return err
		}
	}
	return nil
}

type Field struct {
	Required bool
}

type Message struct {
	Fields map[string]Field
}

func exactCommonField(field *parser.OneofField, n int) *parser.Field {
	return &parser.Field{
		FieldName:    field.FieldName,
		Type:         field.Type,
		FieldNumber:  field.FieldNumber,
		FieldOptions: field.FieldOptions,
		Comments:     field.Comments,
		// For oneof fields, we document the requirement according to the number of fields in the oneof.
		IsRequired: n == 1,
	}
}

func parseField(fs map[string]Field, field *parser.Field) {
	f := Field{}
	if len(field.Comments) > 0 {
		if strings.Contains(field.Comments[0].Lines()[0], "[#do_not_document]") {
			return
		}
	}

	if field.IsRequired || (len(field.FieldOptions) > 0 && !field.IsRepeated && field.Type != "google.protobuf.Duration") {
		f.Required = true
	}

	if len(field.FieldOptions) > 0 {
		for _, option := range field.FieldOptions {
			if option.OptionName == "(validate.rules).repeated" {
				f.Required = true
			}
			if strings.Contains(option.Constant, "required:true") {
				f.Required = true
			}
			if strings.Contains(option.Constant, "ignore_empty:true") {
				f.Required = false
			}
		}
	}
	fs[snakeToCamel(field.FieldName)] = f
}

func parseMapField(fs map[string]Field, field *parser.MapField) {
	f := Field{}
	if len(field.Comments) > 0 {
		if strings.Contains(field.Comments[0].Lines()[0], "[#do_not_document]") {
			return
		}
	}

	if len(field.FieldOptions) > 0 {
		for _, option := range field.FieldOptions {
			if option.OptionName == "(validate.rules).repeated" {
				f.Required = true
			}
			if strings.Contains(option.Constant, "required:true") {
				f.Required = true
			}
			if strings.Contains(option.Constant, "ignore_empty:true") {
				f.Required = false
			}
		}
	}
	fs[snakeToCamel(field.MapName)] = f
}

func parseMessage(ms map[string]Message, msg *parser.Message) {
	m := Message{
		Fields: map[string]Field{},
	}
	for _, body := range msg.MessageBody {
		switch field := body.(type) {
		case *parser.Field:
			parseField(m.Fields, field)
		case *parser.MapField:
			parseMapField(m.Fields, field)
		case *parser.Message:
			parseMessage(ms, field)
		case *parser.Oneof:
			for _, f := range field.OneofFields {
				parseField(m.Fields, exactCommonField(f, len(field.OneofFields)))
			}
		}
	}
	ms[msg.MessageName] = m
}

func readProto(path string) (map[string]Message, error) {
	f, err := os.Open(path)
	// skip Native plugin which doesn't store protobuf file in this repo
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	got, err := protoparser.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf %s: %v", path, err)
	}

	ms := map[string]Message{}
	for _, elem := range got.ProtoBody {
		switch msg := elem.(type) {
		case *parser.Message:
			parseMessage(ms, msg)
		}
	}
	return ms, nil
}

func readDoc(path string, lang string) (map[string]Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	messageName := "Config"
	ms := map[string]Message{
		messageName: {
			Fields: map[string]Field{},
		},
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))

	configTitle := "## Configuration"
	consumerConfigTitle := "## Consumer Configuration"
	trueStr := "True"
	if lang == LangZhHans {
		configTitle = "## 配置"
		consumerConfigTitle = "## 消费者配置"
		trueStr = "是"
	}

	confStarted := false
	for scanner.Scan() {
		text := scanner.Text()
		if confStarted {
			if strings.HasPrefix(text, "## ") && text != consumerConfigTitle {
				confStarted = false
				break
			}

			if strings.HasPrefix(text, "##") {
				if text == consumerConfigTitle {
					messageName = "ConsumerConfig"
				} else {
					ss := strings.Split(text, " ")
					if len(ss) < 2 {
						return nil, errors.New("bad format")
					}
					messageName = ss[1]
				}
				m := Message{
					Fields: map[string]Field{},
				}
				ms[messageName] = m

			} else if strings.HasPrefix(text, "|") && strings.Contains(text, "--") {
				for scanner.Scan() {
					text = scanner.Text()
					if text == "" {
						break
					}

					ss := strings.Fields(text)
					if len(ss) < 6 {
						return nil, errors.New("bad format")
					}
					fieldName := ss[1]
					f := Field{}
					required := ss[5]
					if required == trueStr {
						f.Required = true
					} else {
						f.Required = false
					}
					ms[messageName].Fields[fieldName] = f
				}
			}
		}

		if text == configTitle {
			confStarted = true
		}
	}

	return ms, nil
}

func lintConfigurationByCategory(category string) error {
	err := filepath.Walk(filepath.Join("site", "content", "en", "docs", "reference", category), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".md" || strings.HasSuffix(path, "_index.md") {
			return nil
		}

		name := filepath.Base(path)[:len(filepath.Base(path))-len(ext)]
		goPkgName := strings.ReplaceAll(name, "_", "")
		pb := filepath.Join("types", category, goPkgName, "config.proto")
		ms, err := readProto(pb)
		if err != nil {
			return err
		}

		if ms == nil {
			return nil
		}

		docMs, err := readDoc(path, LangEn)
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(ms, docMs) {
			docMsOut, _ := json.MarshalIndent(docMs, "", "  ")
			msOut, _ := json.MarshalIndent(ms, "", "  ")
			// TODO: also check field type and validation rule
			return fmt.Errorf("mismatched fields in %s:\ndocumented %s\nactual %s", path, docMsOut, msOut)
		}
		return nil
	})
	return err
}

type maturityLevel struct {
	name     string
	maturity string
}

func getFeatureMaturityLevel(category string) ([]maturityLevel, error) {
	res := []maturityLevel{}
	err := filepath.Walk(filepath.Join("site", "content", "en", "docs", "reference", category), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".md" || strings.HasSuffix(path, "_index.md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		status := ""
		for scanner.Scan() {
			text := scanner.Text()
			if text == "## Attribute" {
				for i := 0; i < 3; i++ {
					scanner.Scan() // drop table title
				}

				// the Attribute table in registries doc is different
				if category != "registries" {
					scanner.Scan()
					scanner.Scan()
				}
				// the third row is the status
				scanner.Scan()
				ss := strings.Split(scanner.Text(), "|")
				if len(ss) < 3 {
					return fmt.Errorf("status is missing in the `## Attribute` table of %s", path)
				}

				status = strings.ToLower(strings.TrimSpace(ss[2]))
				break
			}
		}

		name := filepath.Base(path)[:len(filepath.Base(path))-len(ext)]
		res = append(res, maturityLevel{
			name:     filenameToPluginName(name),
			maturity: status,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

type FeatureMaturityLevelRecord struct {
	Name              string `yaml:"name"`
	Status            string `yaml:"status"`
	ExperimentalSince string `yaml:"experimental_since"`
	StableSince       string `yaml:"stable_since"`
}

func lintFeatureMaturityLevel() error {
	recordFile := filepath.Join("maintainer", "feature_maturity_level.yaml")
	data, err := os.ReadFile(recordFile)
	if err != nil {
		return err
	}

	var records map[string][]FeatureMaturityLevelRecord
	err = yaml.Unmarshal(data, &records)
	if err != nil {
		return err
	}

	actualRecords := map[string][]FeatureMaturityLevelRecord{}
	for _, category := range Categories {
		actualRecords[category] = []FeatureMaturityLevelRecord{}
		actual, err := getFeatureMaturityLevel(category)
		if err != nil {
			return err
		}

		for _, record := range actual {
			actualRecords[category] = append(actualRecords[category], FeatureMaturityLevelRecord{
				Name:   record.name,
				Status: record.maturity,
			})
		}
	}

	// also check the plugin execution order documented in this file
	type pluginWrapper struct {
		Name string
		plugins.Plugin
	}
	var pluginList []pluginWrapper
	plugins.IteratePluginType(func(name string, p plugins.Plugin) bool {
		pluginList = append(pluginList, pluginWrapper{
			Name:   name,
			Plugin: p,
		})
		return true
	})
	sort.Slice(pluginList, func(i, j int) bool {
		return plugins.ComparePluginOrder(pluginList[i].Name, pluginList[j].Name)
	})

	var recordedOrder []string
	var runtimeOrder []string
	for _, p := range pluginList {
		runtimeOrder = append(runtimeOrder, p.Name+"\n")
	}

	for category, recs := range records {
		for _, record := range recs {
			if record.Status == "experimental" {
				if record.ExperimentalSince == "" {
					return fmt.Errorf("experimental_since of %s %s is missing in %s", category, record.Name, recordFile)
				}
			} else if record.Status == "stable" {
				if record.StableSince == "" {
					return fmt.Errorf("stable_since of %s %s is missing in %s", category, record.Name, recordFile)
				}
			} else {
				return fmt.Errorf("status '%s' of %s %s is invalid in %s", record.Status, category, record.Name, recordFile)
			}

			found := false
			for i, r := range actualRecords[category] {
				if r.Name == record.Name {
					found = true
					if r.Status != record.Status {
						return fmt.Errorf("status of %s %s is mismatched between %s and the documentation. Please update the record in %s.",
							category, record.Name, recordFile, recordFile)
					}
					actualRecords[category] = slices.Delete(actualRecords[category], i, i+1)
					break
				}
			}
			if !found {
				return fmt.Errorf("feature maturity record of %s %s is missing in the documentation", category, record.Name)
			}

			if category == "plugins" {
				name := record.Name + "\n"
				recordedOrder = append(recordedOrder, name)
			}
		}
	}
	for _, category := range Categories {
		if len(actualRecords[category]) > 0 {
			return fmt.Errorf("%s %s is missing in the %s", category, actualRecords[category][0].Name, recordFile)
		}
	}

	diff := difflib.UnifiedDiff{
		A:        recordedOrder,
		B:        runtimeOrder,
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return err
	}
	if text != "" {
		return errors.New("Plugin order is not correct:\n" + text + ". Please fix the order in " + recordFile)
	}

	return nil
}

func main() {
	// change to the root directory so that we don't need to worry about where this tool locates
	os.Chdir("..")

	type linter func() error
	linters := []linter{
		lintConfiguration,
		lintFilename,
		lintSite,
		lintFeatureMaturityLevel,
	}
	for _, linter := range linters {
		err := linter()
		if err != nil {
			panic(err)
		}
	}
}
