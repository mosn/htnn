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
	"time"

	"github.com/jellydator/ttlcache/v3"

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

func (f *File) Mtime() time.Time {
	f.lock.RLock()
	defer f.lock.RUnlock()
	// the returned time.Time should be readonly
	return f.mtime
}

func (f *File) SetMtime(t time.Time) {
	f.lock.Lock()
	f.mtime = t
	f.lock.Unlock()
}

type fs struct {
	cache *ttlcache.Cache[string, os.FileInfo]
}

func newFS(ttl time.Duration) *fs {
	loader := ttlcache.LoaderFunc[string, os.FileInfo](
		func(c *ttlcache.Cache[string, os.FileInfo], key string) *ttlcache.Item[string, os.FileInfo] {
			info, err := os.Stat(key)
			if err != nil {
				logger.Error(err, "reload file info to cache", "file", key)
				return nil
			}
			item := c.Set(key, info, ttlcache.DefaultTTL)
			return item
		},
	)
	cache := ttlcache.New(
		ttlcache.WithTTL[string, os.FileInfo](ttl),
		ttlcache.WithLoader[string, os.FileInfo](loader),
	)
	go cache.Start()

	return &fs{
		cache: cache,
	}
}

var (
	defaultFs = newFS(10 * time.Second)
)

func IsChanged(files ...*File) bool {
	for _, file := range files {
		changed := defaultFs.isChanged(file)
		if changed {
			return true
		}
	}
	return false
}

func (f *fs) isChanged(file *File) bool {
	item := f.cache.Get(file.Name)
	if item == nil {
		// As a protection, failed to fetch the real file means file not changed
		return false
	}

	return file.Mtime().Before(item.Value().ModTime())
}

func (f *fs) Stat(path string) (*File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	f.cache.Set(path, info, ttlcache.DefaultTTL)

	return &File{
		Name:  path,
		mtime: info.ModTime(),
	}, nil
}

func Stat(path string) (*File, error) {
	return defaultFs.Stat(path)
}

func Update(files ...*File) bool {
	for _, file := range files {
		if !defaultFs.update(file) {
			return false
		}
	}
	return true
}

func (f *fs) update(file *File) bool {
	item := f.cache.Get(file.Name)
	if item == nil {
		return false
	}

	file.SetMtime(item.Value().ModTime())
	return true
}
