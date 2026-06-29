package cmd

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

/*
parseCacheControl takes in the headers in the format recieved from web
and does the following operations on it:
 1. Splits the headers based on comma.
 2. For every header, it splits it based on "=" and stores in a temproary array.
 3. If the header :
    3.1. Does not have a key, intialize it with empty string in map as value.
    3.2. Has a key, store it in key value pair in map.
 4. Returns the map.
*/
func parseCacheControl(headers http.Header) map[string]string {

	cacheHeaders := make(map[string]string)
	rawHeaders := strings.Split(headers.Get("Cache-Control"), ",")

	for i, header := range rawHeaders {
		rawHeaders[i] = strings.TrimSpace(header)
		tempArr := strings.Split(rawHeaders[i], "=")
		if len(tempArr) == 1 {
			cacheHeaders[tempArr[0]] = ""
		} else {
			cacheHeaders[tempArr[0]] = tempArr[1]
		}
	}

	return cacheHeaders
}

/*
ttlFromDirectives takes in the map made by parseCacheControl.
Checks if it has:
 1. s-maxage header.
 2. max-age header.

If it has either of those parameters,
the time proxy caches the request response is taken from it.
If not, we cache the response for 5 minutes.
*/
func ttlFromDirectives(cc map[string]string) time.Duration {

	if value, ok := cc["s-maxage"]; ok {
		if duration, err := strconv.Atoi(value); err == nil {
			return time.Duration(duration) * time.Second
		}
	} else if value, ok := cc["max-age"]; ok {
		if duration, err := strconv.Atoi(value); err == nil {
			return time.Duration(duration) * time.Second
		}
	}
	return 5 * time.Minute
}

/*
cacheDecision evaluates whether a response should be cached and determines its TTL.

It checks the following conditions in order:
  - Status code must be 2xx (200–299).
  - Request must not contain an Authorization header (authenticated responses are not cached).
  - Response must not contain a Set-Cookie header (user-specific responses are not cached).
  - Cache-Control directives are checked:
  - "no-store" and "private" prevent caching entirely.
  - Falls back to 5 minutes if no Cache-Control header is present.

Returns:
  - bool: true if the response should be cached, false otherwise.
  - time.Duration: how long the response should remain in the cache.
*/
func cacheDecision(statusCode int, req *http.Request, respHeaders http.Header) (bool, time.Duration) {

	if (statusCode < 200 || statusCode >= 300) ||
		(req.Header.Get("Authorization") != "") || (respHeaders.Get("Set-Cookie") != "") {
		return false, 0
	}

	cacheMap := parseCacheControl(respHeaders)

	_, noStoreExists := cacheMap["no-store"]
	_, privateExists := cacheMap["private"]
	if noStoreExists || privateExists {
		return false, 0
	}

	timeToCache := ttlFromDirectives(cacheMap)

	return true, timeToCache
}
