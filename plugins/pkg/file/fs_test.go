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
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWatcher is a mock implementation of fsnotify.Watcher
type MockWatcher struct {
	mock.Mock
}

func (m *MockWatcher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestFileIsChanged(t *testing.T) {
	var wg sync.WaitGroup
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
		defaultFsnotify.mu.Lock()
		i = 5
		defaultFsnotify.mu.Unlock()
	}, file)
	assert.Nil(t, err)
	tmpfile.Write([]byte("bls"))
	tmpfile.Sync()
	wg.Wait()

	err = WatchFiles(func() {}, nil)
	assert.Error(t, err, "file pointer cannot be nil")

	defaultFsnotify.mu.Lock()
	assert.Equal(t, 5, i)
	defaultFsnotify.mu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	assert.NoError(t, err)
	defer watcher.Close()
	fs := &Fsnotify{
		WatchedFiles: make(map[string]struct{}),
	}
	tmpDir := filepath.Dir(file.Name)
	fs.WatchedFiles[tmpDir] = struct{}{}
	err = watcher.Add(tmpDir)
	assert.NoError(t, err)

	// check whether onChange is called
	onChangeCalled := false
	onChange := func() {
		onChangeCalled = true
	}

	go fs.watchFiles(onChange, watcher, tmpDir)
	tmpFile, err := os.CreateTemp(tmpDir, "testfile")
	assert.NoError(t, err)
	defer tmpFile.Close()

	time.Sleep(500 * time.Millisecond)
	watcher.Close()
	time.Sleep(500 * time.Millisecond)

	_, exists := fs.WatchedFiles[tmpDir]

	assert.True(t, exists)
	assert.True(t, onChangeCalled)

	err = WatchFiles(func() {}, file, nil)
	assert.Error(t, err, "file pointer cannot be nil")
}

func TestClose(t *testing.T) {
	dir := "./"
	mockWatcher := new(MockWatcher)

	mockWatcher.On("Close").Return(nil)

	defaultfsnotify := struct {
		WatchedFiles map[string]bool
	}{
		WatchedFiles: map[string]bool{dir: true},
	}

	f := struct {
		mu sync.Mutex
	}{}

	func(w *MockWatcher) {
		defer func(w *MockWatcher) {
			f.mu.Lock()
			defer f.mu.Unlock()
			delete(defaultfsnotify.WatchedFiles, dir)
			err := w.Close()
			if err != nil {
				t.Errorf("failed to close fsnotify watcher: %v", err)
			}
		}(w)
	}(mockWatcher)

	assert.NotContains(t, defaultFsnotify.WatchedFiles, dir)

	mockWatcher.AssertExpectations(t)
}
