package cmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

/* startProxy creates and starts the HTTP reverse proxy server.

It takes three arguments:
   - port:      the port the proxy will listen on (e.g.: "8000")
   - urlString: the origin server URL to forward requests to (e.g.: "https://httpbin.org")
   - cacheSize: maximum number of entries the LRU cache can hold

It sets up a Rewrite function on the ReverseProxy that:
  - Rewrites the incoming request URL to point at the origin
  - Sets the Host header on the outbound request
  - Forwards X-Forwarded-* headers so the origin sees the real client

It then creates a CacheManager, the core caching layer and registers two routes via an HTTP mux:
  - "/" — routes all requests through cacheMiddleware (caching + proxy)
  - "/admin/clear-cache" — clears the cache directly, bypassing the proxy

Finally, it starts listening on the specified port. */

func startProxy(port string, urlString string, cacheSize int) {

	port = ":" + port

	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(req *httputil.ProxyRequest) {
			req.SetURL(parsedUrl)
			req.Out.Host = req.In.Host
			req.SetXForwarded()
		},
	}

	cacheManager := NewCacheManager(cacheSize)

	mux := http.NewServeMux()
	mux.Handle("/", cacheMiddleware(proxy, cacheManager))
	mux.HandleFunc("/admin/clear-cache", func(w http.ResponseWriter, r *http.Request) {
		clearCache(cacheManager)
		w.Write([]byte("Cache cleared"))
	})

	fmt.Println("Server is running on port ", port)
	er := http.ListenAndServe(port, mux)
	if er != nil {
		fmt.Println("Error: ", er)
		os.Exit(1)
	}
}
