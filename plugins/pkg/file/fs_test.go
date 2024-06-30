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
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

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

	file := &File{Name: tmpfile.Name()}
	err := WatchFiles(func() {
		wg.Add(1)
		defer wg.Done()
		defaultFsnotify.mu.Lock()
		i = 2
		defaultFsnotify.mu.Unlock()
	}, file)

	assert.Nil(t, err)
	tmpfile.Write([]byte("bls"))
	tmpfile.Sync()
	wg.Wait()

	_ = WatchFiles(func() {
		wg.Add(1)
		defer wg.Done()
		defaultFsnotify.mu.Lock()
		i = 1
		defaultFsnotify.mu.Unlock()
	}, file)
	tmpfile.Sync()
	wg.Wait()

	err = WatchFiles(func() {}, nil)
	assert.Error(t, err, "file pointer cannot be nil")

	filename := "my_file.txt"
	content := "Hello, World!"

	f, err := os.Create(filename)
	fi := &File{Name: f.Name()}
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer f.Close()

	_ = WatchFiles(func() {
		wg.Add(1)
		defer wg.Done()
		defaultFsnotify.mu.Lock()
		i = 3
		defaultFsnotify.mu.Unlock()
	}, fi)
	_, _ = f.WriteString(content)

	_ = os.Remove(filename)
	f, _ = os.Create(filename)

	defer f.Close()

	_, _ = f.WriteString("New content for the file.")
	_ = os.Remove(filename)

	defaultFsnotify.mu.Lock()
	assert.Equal(t, 2, i)
	defaultFsnotify.mu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	assert.NoError(t, err)
	defer watcher.Close()
	fs := &Fsnotify{
		WatchedFiles: make(map[string]struct{}),
	}
	tmpDir, err := ioutil.TempDir("", "watch_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
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

	fs.mu.Lock()
	_, exists := fs.WatchedFiles[tmpDir]
	fs.mu.Unlock()
	assert.True(t, exists, "WatchedFiles should be updated")
	assert.True(t, onChangeCalled, "onChange should be called")
}

func TestStat(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte("hello world")); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	statFile, err := Stat(tmpfile.Name())
	assert.NoError(t, err, "Stat() should not return error")

	assert.Equal(t, tmpfile.Name(), statFile.Name, "Stat() Name should match")
	assert.False(t, statFile.mtime.IsZero(), "Stat() mtime should be non-zero")

	nonExistentFilePath := "./nonexistentfile.txt"
	_, err = Stat(nonExistentFilePath)

	assert.Error(t, err, "Stat should return error for non-existent file")
	assert.True(t, os.IsNotExist(err), "Error should indicate non-existent file")
}
