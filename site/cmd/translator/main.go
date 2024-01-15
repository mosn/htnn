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
var fromLocale = flag.String("from", "en", "the locale code that we translate from")
var toLocale = flag.String("to", "en", "the locale code that we translate to")

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

	fromLang := language.LookupLanguage(*fromLocale)
	if fromLang == nil {
		panic(fmt.Sprintf("language %s not supported", *fromLocale))
	}

	toLang := language.LookupLanguage(*toLocale)
	if toLang == nil {
		panic(fmt.Sprintf("language %s not supported", *toLocale))
	}

	b, err := os.ReadFile(*inputFile)
	mustNoErr(err)
	name := filepath.Base(*inputFile)
	text := "+++" + name + "\n" + string(b) + "\n+++\n"

	t, err := template.New("prompt").Parse(prompt)
	mustNoErr(err)

	glossary := toLang.Glossary()
	if *toLocale == "en" {
		glossary = fromLang.Glossary()
		for i, piece := range glossary {
			glossary[i][0], glossary[i][1] = piece[1], piece[0]
		}
	}

	data := struct {
		InputText string
		Glossary  [][2]string
		Rules     []string
		SrcName   string
		DstName   string
		SrcDemo   string
		DstDemo   string
	}{
		InputText: text,
		Glossary:  glossary,
		Rules:     toLang.Rules(),
		SrcName:   fromLang.Name(),
		DstName:   toLang.Name(),
		SrcDemo:   fromLang.Demo(),
		DstDemo:   toLang.Demo(),
	}

	err = t.Execute(os.Stdout, data)
	mustNoErr(err)
}
