package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

type CacheItem struct {
	responseBody []byte
	headers      http.Header
	statusCode   int
	expiration   time.Time
}

type CacheManager struct {
	storage map[string]CacheItem
	mu      sync.RWMutex
}

type ResponseInterceptor struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func NewCacheManager() *CacheManager {
	return &CacheManager{
		storage: make(map[string]CacheItem),
	}
}

func (cMan *CacheManager) Get(key string) (*CacheItem, bool) {
	cMan.mu.RLock()
	defer cMan.mu.RUnlock()

	item, found := cMan.storage[key]
	if !found {
		return nil, false
	}
	if time.Now().After(item.expiration) {
		delete(cMan.storage, key)
		return nil, false
	}
	return &item, true
}

func (cMan *CacheManager) Set(key string, item *CacheItem) {
	cMan.mu.Lock()
	defer cMan.mu.Unlock()
	cMan.storage[key] = *item
}

func NewResponseInterceptor(w http.ResponseWriter) *ResponseInterceptor {
	return &ResponseInterceptor{
		ResponseWriter: w,
		body:           new(bytes.Buffer),
		statusCode:     http.StatusOK,
	}
}

func (ri *ResponseInterceptor) WriteHeader(statusCode int) {
	ri.statusCode = statusCode
	ri.ResponseWriter.WriteHeader(statusCode)
}

func (ri *ResponseInterceptor) Write(p []byte) (int, error) {
	ri.body.Write(p)
	return ri.ResponseWriter.Write(p)
}

func cacheMiddleware(proxy *httputil.ReverseProxy, cache *CacheManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		if r.Method != http.MethodGet {
			proxy.ServeHTTP(w, r)
			return
		}

		cacheKey := r.URL.String()
		
		if item, found := cache.Get(cacheKey); found {
			log.Printf("Cache hit for %s", cacheKey)
			w.Header().Set("X-Cache", "HIT")
			for k, v := range item.headers {
				w.Header()[k] = v
			}
			w.WriteHeader(item.statusCode)
			w.Write(item.responseBody)
			return
		}

		fmt.Println("CACHE MISS for", cacheKey)

		interceptor := NewResponseInterceptor(w)
		interceptor.Header().Set("X-Cache", "MISS")
		proxy.ServeHTTP(interceptor, r)

		if interceptor.statusCode <= 300 && interceptor.statusCode >= 200 {
			cacheItem := &CacheItem{
				responseBody: interceptor.body.Bytes(),
				headers:      interceptor.Header(),
				statusCode:   interceptor.statusCode,
				expiration:   time.Now().Add(5 * time.Minute),
			}
			cache.Set(cacheKey, cacheItem)
		}

	}
}

func clearCache(cache *CacheManager) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.storage = make(map[string]CacheItem)
}