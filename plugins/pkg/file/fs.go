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
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("file")
)

type File struct {
	lock sync.RWMutex

	Name  string
	mtime time.Time
}

type Fsnotify struct {
	Watcher *fsnotify.Watcher
}

func newFsnotify() (fs *Fsnotify) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "create watcher failed")
		return
	}
	fs = &Fsnotify{
		Watcher: watcher,
	}
	return
}

var (
	defaultFsnotify = newFsnotify()
)

func Update(onChange func(), files ...*File) (err error) {
	err = WatchFiles(onChange, files...)
	return
}

func WatchFiles(onChange func(), files ...*File) (err error) {
	if len(files) < 1 {
		err = errors.New("must specify at least one file to watch")
		return
	}

	watcher := newFsnotify().Watcher
	if err != nil {
		return
	}

	// Add files to watcher.
	for _, file := range files {
		go defaultFsnotify.watchFiles(onChange, watcher, file)
	}

	return
}

func (f *Fsnotify) watchFiles(onChange func(), w *fsnotify.Watcher, files *File) {
	defer func(w *fsnotify.Watcher) {
		logger.Info("stop watch files" + files.Name)
		err := w.Close()
		if err != nil {
			logger.Error(err, "failed to close fsnotify watcher")
		}
	}(w)
	err := w.Add(files.Name)
	if err != nil {
		logger.Error(err, "add file to watcher failed")
	}
	logger.Info("start watch files" + files.Name)
	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			logger.Info(fmt.Sprintf("event: %v", event))
			onChange()
			return
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
