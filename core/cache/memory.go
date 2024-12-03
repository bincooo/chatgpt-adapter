package cache

import (
	"context"
	"strings"
	"time"

	"chatgpt-adapter/core/common/inited"
	"chatgpt-adapter/core/gin/model"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/iocgo/sdk/env"

	gocacheStore "github.com/eko/gocache/store/go_cache/v4"
	gocache "github.com/patrickmn/go-cache"
)

type Manager[T any] struct {
	cache *cache.Cache[T]
}

var (
	toolTasksCacheManager *Manager[[]model.Keyv[string]]
)

func init() {
	inited.AddInitialized(func(_ *env.Environment) {
		client := gocache.New(5*time.Minute, 5*time.Minute)
		cacheStore := gocacheStore.NewGoCache(client)
		toolTasksCacheManager = &Manager[[]model.Keyv[string]]{
			cache.New[[]model.Keyv[string]](cacheStore),
		}
	})
}

func ToolTasksCacheManager() *Manager[[]model.Keyv[string]] {
	return toolTasksCacheManager
}

func (cacheManager *Manager[T]) SetValue(key string, value T) error {
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return cacheManager.cache.Set(timeout, key, value, store.WithExpiration(120*time.Second))
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
