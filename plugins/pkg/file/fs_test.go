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
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		changed bool
	)
	watcher, err := fsnotify.NewWatcher()
	defer func(watcher *fsnotify.Watcher) {
		err := watcher.Close()
		if err != nil {
			t.Errorf("close watcher err:%v", err)
		}
	}(watcher)

	assert.Nil(t, err)

	tmpfile, _ := os.CreateTemp("./", "example")

	file, err := Stat(tmpfile.Name(), watcher)

	assert.NoError(t, err)
	assert.Equal(t, tmpfile.Name(), file.Name)

	tmpDir := filepath.Dir(tmpfile.Name())
	storeWatchedFiles.lock.RLock()
	_, exists := storeWatchedFiles.WatchedFiles[tmpDir]
	storeWatchedFiles.lock.RUnlock()
	assert.True(t, exists)

	err = WatchFiles(func() {
		mu.Lock()
		changed = true
		mu.Unlock()
	}, file)
	assert.Nil(t, err)
	tmpfile.Write([]byte("bls"))
	tmpfile.Sync()
	wg.Wait()

	err = WatchFiles(func() {}, nil)

	assert.Error(t, err, "file pointer cannot be nil")

	mu.Lock()
	assert.True(t, changed)
	mu.Unlock()

	err = os.Remove(tmpfile.Name())
	assert.Nil(t, err)
}
