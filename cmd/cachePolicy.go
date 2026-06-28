package cmd

import (
	"fmt"
	"net/http"
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

	for k, v := range cacheHeaders {
		fmt.Printf("Key: %s, Value: %s\n", k, v)
	}

	return cacheHeaders
}

func ttlFromDirectives(cc map[string]string) time.Duration {

	return 5 * time.Minute
}

func cacheDecision(statusCode int, req *http.Request, respHeaders http.Header) (bool, time.Duration) {

	parseCacheControl(respHeaders)
	return true, 5 * time.Minute
}
