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

/*
Holds a cached HTTP response.
*/
type CacheItem struct {
	responseBody []byte
	headers      http.Header
	statusCode   int
	expiration   time.Time
}

/*
The storage map holds all the cache item (inside a list).
This along with list helps implement Least Recently Used(LRU).
Mutex is for locking while working on map. Maxsize is maximum size of the cache.
*/
type CacheManager struct {
	storage map[string]*list.Element
	list    *list.List
	mu      sync.Mutex
	maxSize int
}

/*
Wrapper for storing key value pairs in linked list.
Key is used to delete an item from the list.
*/
type CacheEntry struct {
	key  string
	item CacheItem
}

/*
Captures the origin's response for caching while still streaming it to the client.
*/
type ResponseInterceptor struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

/*
Creates and returns a new instance of cache manager.
*/
func NewCacheManager(maxSize int) *CacheManager {
	return &CacheManager{
		storage: make(map[string]*list.Element),
		list:    list.New(),
		maxSize: maxSize,
	}
}

/*
Gets the cacheItem from the cachemanager's storage.
Checks the cacheItem's expiration.

	If expired, remove it from the list and delete the corresponding key from the storage map.

Move the item to front of the list and return it.
*/
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

/*
The set function takes in:

	key: A key to for storage map.
	item: An item to cache,

The function first searches in the cache if the key already exists.

	If so, move it to front of list and return.

If not, it checks if the list is already at max capacity.

	If it is, take the element from the back of the list and remove it from list.
	Delete the corresponding key from the storage map.

Makes the new entry and stores it in cache.
*/
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

/*
Creates and returns a new instance of response interceptor.
*/
func NewResponseInterceptor(w http.ResponseWriter) *ResponseInterceptor {
	return &ResponseInterceptor{
		ResponseWriter: w,
		body:           new(bytes.Buffer),
		statusCode:     http.StatusOK,
	}
}

/*
WriteHeader captures the status code returned by the origin server
then passes it through to the underlying ResponseWriter.
This status code is later used to decide whether to cache the response.
*/
func (ri *ResponseInterceptor) WriteHeader(statusCode int) {
	ri.statusCode = statusCode
	ri.ResponseWriter.WriteHeader(statusCode)
}

/*
Write copies the response body into the internal buffer for caching
while simultaneously writing it to the actual client response.
The buffered bytes are later stored in the cache on a cache miss.
*/
func (ri *ResponseInterceptor) Write(p []byte) (int, error) {
	ri.body.Write(p)
	return ri.ResponseWriter.Write(p)
}

/*
cacheMiddleware returns an HTTP handler that wraps the reverse proxy with caching.

On each request:
  - Non-GET requests are proxied directly without caching.
  - GET requests first check the cache by URL key.
  - Cache hit: returns the cached response with X-Cache: HIT.
  - Cache miss: forwards the request to the origin via the proxy, then caches the response
    if it meets all conditions (2xx status, no Authorization header on the request, no Set-Cookie header on the response).
*/
func cacheMiddleware(proxy *httputil.ReverseProxy, cache *CacheManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodGet {
			proxy.ServeHTTP(w, r)
			return
		}

		cacheKey := r.URL.String()

		if item, found := cache.Get(cacheKey); found {
			log.Printf("CACHE HIT for %s", cacheKey)
			for k, v := range item.headers {
				w.Header()[k] = v
			}
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(item.statusCode)
			w.Write(item.responseBody)
			return
		}

		log.Printf("CACHE MISS for %s", cacheKey)

		interceptor := NewResponseInterceptor(w)
		interceptor.Header().Set("X-Cache", "MISS")
		proxy.ServeHTTP(interceptor, r)

		shouldCache, timeToCacheFor := cacheDecision(interceptor.statusCode, r, interceptor.Header())

		if shouldCache {
			cacheItem := &CacheItem{
				responseBody: interceptor.body.Bytes(),
				headers:      interceptor.Header(),
				statusCode:   interceptor.statusCode,
				expiration:   time.Now().Add(timeToCacheFor),
			}
			cache.Set(cacheKey, cacheItem)
		}
	}
}

/*
Reinitializes the list and storage to clear the cache.
*/
func clearCache(cache *CacheManager) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.list.Init()
	cache.storage = make(map[string]*list.Element)
}
