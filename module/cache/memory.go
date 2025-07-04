package cache

import (
	"adapter/module/common"
	"adapter/module/env"
	"adapter/module/fiber/model"
	"context"
	"strings"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"

	gocacheStore "github.com/eko/gocache/store/go_cache/v4"
	gocache "github.com/patrickmn/go-cache"
)

type Manager[T any] struct {
	cache *cache.Cache[T]
}

var (
	toolTasksCacheManager *Manager[[]model.Record[string, any]]
	windsurfCacheManager  *Manager[string]
	bingCacheManager      *Manager[string]
	cursorCacheManager    *Manager[string]
	qodoCacheManager      *Manager[string]
)

func init() {
	common.AddInitialized(func(_ *env.Environ) {
		client := gocache.New(5*time.Minute, 5*time.Minute)
		toolTasksCacheManager = &Manager[[]model.Record[string, any]]{
			cache.New[[]model.Record[string, any]](gocacheStore.NewGoCache(client)),
		}

		client = gocache.New(5*time.Minute, 5*time.Minute)
		windsurfCacheManager = &Manager[string]{
			cache.New[string](gocacheStore.NewGoCache(client)),
		}

		client = gocache.New(5*time.Minute, 5*time.Minute)
		bingCacheManager = &Manager[string]{
			cache.New[string](gocacheStore.NewGoCache(client)),
		}

		client = gocache.New(5*time.Minute, 5*time.Minute)
		cursorCacheManager = &Manager[string]{
			cache.New[string](gocacheStore.NewGoCache(client)),
		}

		client = gocache.New(5*time.Minute, 5*time.Minute)
		qodoCacheManager = &Manager[string]{
			cache.New[string](gocacheStore.NewGoCache(client)),
		}
	})
}

func GetToolTasksCacheManager() *Manager[[]model.Record[string, any]] {
	return toolTasksCacheManager
}

func GetWindsurfCacheManager() *Manager[string] {
	return windsurfCacheManager
}

func GetBingCacheManager() *Manager[string] {
	return bingCacheManager
}

func GetCursorCacheManager() *Manager[string] {
	return cursorCacheManager
}

func GetQodoCacheManager() *Manager[string] {
	return qodoCacheManager
}

func (cacheManager *Manager[T]) SetValue(key string, value T) error {
	return cacheManager.SetWithExpiration(key, value, 120*time.Second)
}

func (cacheManager *Manager[T]) SetWithExpiration(key string, value T, expir time.Duration) error {
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return cacheManager.cache.Set(timeout, key, value, store.WithExpiration(expir))
}

func (cacheManager *Manager[T]) GetValue(key string) (value T, err error) {
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const errorMessage = "value not found"
	value, err = cacheManager.cache.Get(timeout, key)
	if err != nil && strings.Contains(err.Error(), errorMessage) {
		err = nil
		return
	}
	return
}

func (cacheManager *Manager[T]) Delete(key string) error {
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return cacheManager.cache.Delete(timeout, key)
}
