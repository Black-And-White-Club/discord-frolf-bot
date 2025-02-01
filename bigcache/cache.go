package cache

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
)

type CacheInterface interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

type Cache struct {
	bigCache *bigcache.BigCache
}

func NewCache(ctx context.Context) (*Cache, error) {
	config := bigcache.DefaultConfig(10 * time.Minute)
	bigCache, err := bigcache.New(ctx, config)
	if err != nil {
		return nil, err
	}
	return &Cache{bigCache: bigCache}, nil
}

func (c *Cache) Set(key string, value []byte) error {
	return c.bigCache.Set(key, value)
}

func (c *Cache) Get(key string) ([]byte, error) {
	return c.bigCache.Get(key)
}

func (c *Cache) Delete(key string) error {
	return c.bigCache.Delete(key)
}
