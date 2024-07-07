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
	"path/filepath"
	"sync"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestFileIsChanged(t *testing.T) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	i := 4

	tmpfile, _ := os.CreateTemp("./", "example")
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Logf("%v", err)
		}
	}(tmpfile.Name())
	file, err := Stat(tmpfile.Name())

	assert.NoError(t, err)
	assert.Equal(t, tmpfile.Name(), file.Name)
	err = WatchFiles(func() {
		wg.Add(1)
		defer wg.Done()
		mu.Lock()
		i = 5
		mu.Unlock()
	}, file)
	assert.Nil(t, err)
	tmpfile.Write([]byte("bls"))
	tmpfile.Sync()
	wg.Wait()

	err = WatchFiles(func() {}, nil)
	assert.Error(t, err, "file pointer cannot be nil")

	mu.Lock()
	assert.Equal(t, 5, i)
	mu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	assert.NoError(t, err)
	defer watcher.Close()
	fs := &Fsnotify{
		WatchedFiles: make(map[string]struct{}),
		Watcher:      watcher,
	}
	tmpfile, err = os.CreateTemp("/tmp", "test")
	assert.Nil(t, err)
	defer os.Remove(tmpfile.Name())
	tmpDir := filepath.Dir(tmpfile.Name())
	fs.WatchedFiles[tmpDir] = struct{}{}
	err = fs.AddFiles(tmpDir)
	assert.NoError(t, err)

	onChangeCalled := false
	onChange := func() {
		onChangeCalled = true
	}

	go fs.watchFiles(onChange, fs.Watcher, tmpDir)

	_, exists := fs.WatchedFiles[tmpDir]

	assert.True(t, exists)
	assert.False(t, onChangeCalled)

}
