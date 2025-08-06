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

package contentbuffer

import (
	"fmt"
	"unicode/utf8"
)

type CharCounter interface {
	Count(data []byte) int
	DecodeChar(data []byte) (r rune, size int, err error)
	TailStartIndex(data []byte, n int) int
}

type Utf8RuneCounter struct{}

func (Utf8RuneCounter) Count(data []byte) int { return utf8.RuneCount(data) }

func (Utf8RuneCounter) DecodeChar(data []byte) (rune, int, error) {
	r, size := utf8.DecodeRune(data)
	if r == utf8.RuneError && size == 1 {
		return r, size, fmt.Errorf("invalid utf8 encoding")
	}
	return r, size, nil
}

func (Utf8RuneCounter) TailStartIndex(data []byte, n int) int {
	if n <= 0 {
		return len(data)
	}
	count := 0
	i := len(data)
	for i > 0 && count < n {
		_, size := utf8.DecodeLastRune(data[:i])
		i -= size
		count++
	}
	return i
}
