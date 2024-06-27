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

package file

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileIsChanged(t *testing.T) {
	i := 1
	tmpfile, _ := os.CreateTemp("./", "example")
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Logf("%v", err)
		}
	}(tmpfile.Name())

	file := &File{Name: tmpfile.Name()}
	_ = WatchFiles(func() {
		i = 2
	}, file)
	time.Sleep(1 * time.Millisecond)
	tmpfile.Write([]byte("bls"))
	tmpfile.Sync()
	assert.Equal(t, 2, i)

	_ = WatchFiles(func() {
		i = 1
	}, file)
	time.Sleep(1 * time.Millisecond)
	tmpfile.Sync()
	assert.Equal(t, 2, i)

	err := WatchFiles(func() {})
	assert.Equal(t, err.Error(), "must specify at least one file to watch", "Expected error message does not match")

}
