package cmd

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
