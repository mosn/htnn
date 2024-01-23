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
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/waigani/diffparser"

	"mosn.io/htnn/site/cmd/translator/language"
)

var (
	//go:embed prompt.txt
	prompt string
)

var commit = flag.String("c", "HEAD", "the base commit to compare with when doing incremental translation")
var inputFile = flag.String("f", "", "the file to translate")
var fromLocale = flag.String("from", "en", "the locale code that we translate from")
var toLocale = flag.String("to", "en", "the locale code that we translate to")

func mustNoErr(err error) {
	if err != nil {
		panic(err)
	}
}

type Chunk struct {
	id    int
	text  string
	start int
	end   int
}

type Range struct {
	start int
	end   int
}

func findAffectedChunks(ranges []Range, chunks []Chunk) []Chunk {
	affectedChunks := []Chunk{}
	i, j := 0, 0
	for i < len(ranges) && j < len(chunks) {
		r := ranges[i]
		c := chunks[j]
		if r.end < c.start {
			i++
		} else if r.start > c.end {
			j++
		} else {
			affectedChunks = append(affectedChunks, c)
			j++
		}
	}
	return affectedChunks
}

func main() {
	flag.Parse()

	if *inputFile == "" {
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
	text := string(b)

	diff := ""
	outputFile := *inputFile
	outputFile = strings.Replace(outputFile, strings.ToLower(*fromLocale), strings.ToLower(*toLocale), 1)
	if _, err := os.Stat(outputFile); err == nil {
		cmdline := fmt.Sprintf("git diff %s -- %s", *commit, *inputFile)
		cmd := strings.Fields(cmdline)
		runDiff := exec.Command(cmd[0], cmd[1:]...)
		b, err = runDiff.Output()
		mustNoErr(err)

		diff = string(b)
	}

	if diff == "" {
		text = "+++" + name + "\n" + text + "\n+++\n"
	} else {
		lines := strings.Split(text, "\n")
		start := 0

		chunks := []Chunk{}
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				chunks = append(chunks, Chunk{
					id:    len(chunks),
					text:  strings.Join(lines[start:i], "\n"),
					start: start,
					end:   i,
				})
				start = i + 1
			}
		}

		parsed, _ := diffparser.Parse(diff)
		hunks := parsed.Files[0].Hunks
		ranges := []Range{}
		for _, h := range hunks {
			n := len(h.NewRange.Lines)
			startHunk := 0
			for i := 0; i < n; i++ {
				li := h.NewRange.Lines[i]
				if startHunk == 0 {
					if li.Mode == diffparser.ADDED {
						startHunk = li.Number
					}
				} else if li.Mode != diffparser.ADDED {
					ranges = append(ranges, Range{
						start: startHunk,
						end:   li.Number - 1,
					})
					startHunk = 0
				}
			}

			if startHunk != 0 {
				ranges = append(ranges, Range{
					start: startHunk,
					end:   h.NewRange.Lines[n-1].Number,
				})
			}
		}

		affectedChunks := findAffectedChunks(ranges, chunks)
		out := strings.Builder{}
		// must have at least one affected chunk
		out.WriteString(affectedChunks[0].text)
		for i, c := range affectedChunks[1:] {
			if c.id == affectedChunks[i].id+1 {
				out.WriteString("\n\n")
			} else {
				// this acts as separator
				out.WriteString("\n...\n")
			}
			out.WriteString(c.text)
		}

		// pretend as a new doc
		text = fmt.Sprintf("+++%s\n", name) + out.String() + "\n+++\n"
	}

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
