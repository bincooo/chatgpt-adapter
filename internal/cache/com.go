package cache

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/pkg"
	"context"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	gocacheStore "github.com/eko/gocache/store/go_cache/v4"
	gocache "github.com/patrickmn/go-cache"
	"strings"
	"time"
)

var (
	toolTasksCacheManager *cache.Cache[[]pkg.Keyv[string]]
)

func init() {
	common.AddInitialized(func() {
		client := gocache.New(5*time.Minute, 5*time.Minute)
		cacheStore := gocacheStore.NewGoCache(client)
		toolTasksCacheManager = cache.New[[]pkg.Keyv[string]](cacheStore)
	})
}

func GetToolTasksCacheManager() *cache.Cache[[]pkg.Keyv[string]] {
	return toolTasksCacheManager
}

func CacheToolTasksValue(key string, value []pkg.Keyv[string]) error {
	cacheManager := GetToolTasksCacheManager()
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return cacheManager.Set(timeout, key, value, store.WithExpiration(120*time.Second))
}

func GetToolTasksCache(key string) (value []pkg.Keyv[string], err error) {
	const errorMessage = "value not found"
	cacheManager := GetToolTasksCacheManager()
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	value, err = cacheManager.Get(timeout, key)
	if err != nil && strings.Contains(err.Error(), errorMessage) {
		return nil, nil
	}

	return
}
