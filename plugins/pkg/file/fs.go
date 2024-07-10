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
	"errors"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"

	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("file")
)

type File struct {
	Name    string
	Watcher *fsnotify.Watcher
	mu      sync.RWMutex
}

type StoreWatchedFiles struct {
	WatchedFiles map[string]struct{}
	lock         *sync.RWMutex
}

func newStoreWatcherFiles() *StoreWatchedFiles {
	return &StoreWatchedFiles{
		WatchedFiles: make(map[string]struct{}),
		lock:         &sync.RWMutex{},
	}
}

var storeWatchedFiles = newStoreWatcherFiles()

func WatchFiles(onChanged func(), file *File, otherFiles ...*File) (err error) {
	files := append([]*File{file}, otherFiles...)
	for _, f := range files {
		if f == nil {
			return errors.New("file pointer cannot be nil")
		}
	}

	// Add files to watcher.
	for _, f := range files {
		go watchFiles(onChanged, f)
	}

	return
}

func watchFiles(onChanged func(), file *File) {
	dir := filepath.Dir(file.Name)
	defer func() {
		storeWatchedFiles.lock.Lock()
		defer storeWatchedFiles.lock.Unlock()
		delete(storeWatchedFiles.WatchedFiles, dir)

	}()

	for {
		select {
		case event, ok := <-file.Watcher.Events:
			if !ok {
				return
			}
			logger.Info("file changed: ", "event", event)
			onChanged()
		case err, ok := <-file.Watcher.Errors:
			if !ok {
				return
			}
			logger.Error(err, "error watching files")
		}
	}
}

func AddFiles(file string, w *fsnotify.Watcher) (err error) {
	dir := filepath.Dir(file)

	storeWatchedFiles.lock.RLock()

	if _, exists := storeWatchedFiles.WatchedFiles[dir]; exists {
		storeWatchedFiles.lock.RUnlock()
		return
	}
	storeWatchedFiles.lock.RUnlock()

	storeWatchedFiles.lock.Lock()
	storeWatchedFiles.WatchedFiles[dir] = struct{}{}
	storeWatchedFiles.lock.Unlock()

	err = w.Add(dir)
	return
}
func Stat(file string, w *fsnotify.Watcher) (*File, error) {
	err := AddFiles(file, w)
	return &File{
		Name:    file,
		Watcher: w,
		mu:      sync.RWMutex{},
	}, err
}
