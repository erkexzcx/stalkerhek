package main

import (
	"sync"
	"time"
)

const cacheLifetime = time.Duration(20 * time.Second)

type cache struct {
	Content      []byte
	ContentType  string
	CreationTime time.Time
}

var cacheMap = make(map[string]*cache)
var cacheMapMutex sync.RWMutex

func loadCache(key string) ([]byte, string, bool) {
	cacheMapMutex.RLock()
	defer cacheMapMutex.RUnlock()

	el, exists := cacheMap[key]
	if !exists {
		return nil, "", false
	}
	if time.Since(el.CreationTime) > cacheLifetime {
		return nil, "", false
	}
	return el.Content, el.ContentType, true
}

func saveCache(key string, content *[]byte, contentType *string) {
	cacheMapMutex.Lock()
	defer cacheMapMutex.Unlock()

	cacheMap[key] = &cache{
		Content:      *content,
		ContentType:  *contentType,
		CreationTime: time.Now(),
	}
}

func removeOldCache() {
	cacheMapMutex.Lock()
	defer cacheMapMutex.Unlock()

	for k, el := range cacheMap {
		if time.Since(el.CreationTime) > cacheLifetime {
			delete(cacheMap, k)
		}
	}
}

func parseAndCacheM3U8(c *tvchannel, content *[]byte) {

}
