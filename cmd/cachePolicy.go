package cmd

import (
	"net/http"
	"time"
)

func parseCacheControl(headers http.Header) map[string][]string {

	return make(map[string][]string)
}

func hasDirective(cc map[string][]string, directive string) bool {

	return false
}

func ttlFromDirectives(cc map[string][]string) time.Duration {

	return 5 * time.Minute
}

func cacheDecision(statusCode int, req *http.Request, respHeaders http.Header) (bool, time.Duration) {

	return true, 5 * time.Minute
}
