package cmd

import (
	"bytes"
	"container/list"
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
	storage map[string]*list.Element
	list    *list.List
	mu      sync.Mutex
	maxSize int
}

type CacheEntry struct {
	key  string
	item CacheItem
}

type ResponseInterceptor struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func NewCacheManager(maxSize int) *CacheManager {
	return &CacheManager{
		storage: make(map[string]*list.Element),
		list:    list.New(),
		maxSize: maxSize,
	}
}

func (cMan *CacheManager) Get(key string) (*CacheItem, bool) {
	cMan.mu.Lock()
	defer cMan.mu.Unlock()

	item, found := cMan.storage[key]
	if !found {
		return nil, false
	}
	entryInCache := item.Value.(*CacheEntry)
	if time.Now().After(entryInCache.item.expiration) {
		cMan.list.Remove(item)
		delete(cMan.storage, key)
		return nil, false
	}
	cMan.list.MoveToFront(item)
	return &entryInCache.item, true
}

func (cMan *CacheManager) Set(key string, item *CacheItem) {
	cMan.mu.Lock()
	defer cMan.mu.Unlock()

	if elem, found := cMan.storage[key]; found {
		cMan.list.MoveToFront(elem)
		elem.Value.(*CacheEntry).item = *item
		return
	}

	if cMan.list.Len() >= cMan.maxSize {
		lruElem := cMan.list.Back()
		cMan.list.Remove(lruElem)
		delete(cMan.storage, lruElem.Value.(*CacheEntry).key)
	}

	entry := &CacheEntry{key: key, item: *item}
	newEle := cMan.list.PushFront(entry)
	cMan.storage[key] = newEle
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

		log.Printf("CACHE MISS for %s", cacheKey)

		interceptor := NewResponseInterceptor(w)
		interceptor.Header().Set("X-Cache", "MISS")
		proxy.ServeHTTP(interceptor, r)

		if interceptor.statusCode >= 200 && interceptor.statusCode < 300 {
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
	cache.list.Init()
	cache.storage = make(map[string]*list.Element)
}
