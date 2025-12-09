package db

import (
	"time"

	"github.com/patrickmn/go-cache"
)

func initCache() {
	DB.Cache = cache.New(5*time.Minute, 10*time.Minute)
}

func CacheSet(key string, value interface{}, expires time.Duration) {
	DB.Cache.Set(key, value, expires)
}

func CacheGet(key string) (interface{}, bool) {
	res, ok := DB.Cache.Get(key)
	return res, ok
}

func CacheDelete(key string) {
	DB.Cache.Delete(key)
}
