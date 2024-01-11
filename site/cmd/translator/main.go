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
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"mosn.io/htnn/site/cmd/translator/language"
)

var (
	//go:embed prompt.txt
	prompt string
)

var inputFile = flag.String("f", "", "the file to translate")
var locale = flag.String("l", "zh-Hans", "the locale code that we support")

func mustNoErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()

	if inputFile == nil || *inputFile == "" {
		panic("no input file")
	}

	lang := language.LookupLanguage(*locale)
	if lang == nil {
		panic(fmt.Sprintf("language %s not supported", *locale))
	}

	b, err := os.ReadFile(*inputFile)
	mustNoErr(err)
	name := filepath.Base(*inputFile)
	text := "+++" + name + "\n" + string(b) + "\n+++\n"

	t, err := template.New("prompt").Parse(prompt)
	mustNoErr(err)

	data := struct {
		InputText string
		Name      string
		Glossary  string
		Rules     []string
		Demo      string
	}{
		InputText: text,
		Name:      lang.Name(),
		Glossary:  lang.Glossary(),
		Rules:     lang.Rules(),
		Demo:      lang.Demo(),
	}

	err = t.Execute(os.Stdout, data)
	mustNoErr(err)
}
