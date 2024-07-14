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
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"

	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("file")
)

type File struct {
	Name string
}

type Watcher struct {
	watcher *fsnotify.Watcher
	files   map[string]bool
	mu      sync.Mutex
	dir     map[string]bool
	done    chan struct{}
}

func NewWatcher() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher: w,
		files:   make(map[string]bool),
		done:    make(chan struct{}),
		dir:     make(map[string]bool),
	}, nil
}

func (w *Watcher) AddFiles(files ...*File) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, file := range files {
		if _, exists := w.files[file.Name]; !exists {
			w.files[file.Name] = true
		}
		dir := filepath.Dir(file.Name)
		if _, exists := w.dir[dir]; !exists {
			if err := w.watcher.Add(dir); err != nil {
				return err
			}
			w.dir[dir] = true
		}
	}
	return nil
}

func (w *Watcher) Start(onChanged func()) {
	go func() {
		logger.Info("start watching files")
		for {
			select {
			case event := <-w.watcher.Events:
				if _, exists := w.files[event.Name]; exists {
					logger.Info("file changed: ", "event", event)
					onChanged()
				}
			case err := <-w.watcher.Errors:
				logger.Error(err, "error watching files")
			case <-w.done:
				return
			}
		}
	}()
}

func (w *Watcher) Stop() error {
	logger.Info("stop watcher")
	close(w.done)
	return w.watcher.Close()
}

func Stat(file string) *File {
	return &File{
		Name: file,
	}
}
