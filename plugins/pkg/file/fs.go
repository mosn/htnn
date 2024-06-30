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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("file")
)

type File struct {
	Name  string
	mtime time.Time
}

type Fsnotify struct {
	mu           sync.Mutex
	Watcher      *fsnotify.Watcher
	WatchedFiles map[string]struct{}
}

func newFsnotify() (fs *Fsnotify) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "create watcher failed")
		return
	}

	return &Fsnotify{
		Watcher:      watcher,
		WatchedFiles: make(map[string]struct{}),
	}

}

var (
	defaultFsnotify = newFsnotify()
)

func WatchFiles(onChange func(), file *File, otherFiles ...*File) (err error) {
	files := append([]*File{file}, otherFiles...)
	for _, f := range files {
		if f == nil {
			return errors.New("file pointer cannot be nil")
		}
	}

	watcher := defaultFsnotify.Watcher

	// Add files to watcher.
	for _, f := range files {
		dir := filepath.Dir(f.Name)
		err = defaultFsnotify.AddFiles(dir)
		if err != nil {
			logger.Error(err, "failed to add file")
		}
		if _, exists := defaultFsnotify.WatchedFiles[dir]; exists {
			logger.Info(fmt.Sprintf("File %s is already being watched", f.Name))
			continue
		}
		// 添加到已监听文件的集合
		defaultFsnotify.WatchedFiles[dir] = struct{}{}
		go defaultFsnotify.watchFiles(onChange, watcher, dir)
	}

	return
}

func (f *Fsnotify) AddFiles(dir string) (err error) {
	err = f.Watcher.Add(dir)
	return
}

func (f *Fsnotify) watchFiles(onChange func(), w *fsnotify.Watcher, dir string) {
	defer func(w *fsnotify.Watcher) {
		f.mu.Lock()
		delete(defaultFsnotify.WatchedFiles, dir)
		f.mu.Unlock()
		err := w.Close()
		if err != nil {
			logger.Error(err, "failed to close fsnotify watcher")
		}
	}(w)

	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			logger.Info(fmt.Sprintf("event: %v", event))
			onChange()
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			logger.Error(err, "error watching files")
		}
	}
}

func (f *Fsnotify) Stat(path string) (*File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return &File{
		Name:  path,
		mtime: info.ModTime(),
	}, nil
}

func Stat(path string) (*File, error) {
	return defaultFsnotify.Stat(path)
}
