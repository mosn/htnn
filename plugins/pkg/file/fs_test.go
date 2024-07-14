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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileIsChanged(t *testing.T) {
	changed := false
	wg := sync.WaitGroup{}
	once := sync.Once{}

	watcher, err := NewWatcher()

	assert.Nil(t, err)

	tmpfile, _ := os.CreateTemp("./", "example")

	file := Stat(tmpfile.Name())

	assert.Equal(t, tmpfile.Name(), file.Name)

	err = watcher.AddFiles(file)
	assert.Nil(t, err)
	wg.Add(1)
	watcher.Start(func() {
		once.Do(func() {
			changed = true
			wg.Done()
		})
	})
	assert.Nil(t, err)
	tmpfile.Write([]byte("bls"))
	tmpfile.Sync()

	wg.Wait()
	assert.True(t, changed)

	err = os.Remove(tmpfile.Name())
	assert.Nil(t, err)

	err = watcher.Stop()
	assert.Nil(t, err)
}
