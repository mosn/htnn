package file

import (
	"os"
	"time"

	"github.com/jellydator/ttlcache/v3"

	"mosn.io/moe/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("file")
)

type File struct {
	Name  string
	Mtime time.Time
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

	return file.Mtime.Before(item.Value().ModTime())
}

func (f *fs) Stat(path string) (*File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	f.cache.Set(path, info, ttlcache.DefaultTTL)

	return &File{
		Name:  path,
		Mtime: info.ModTime(),
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

	file.Mtime = item.Value().ModTime()
	return true
}
