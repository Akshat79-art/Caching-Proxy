package cmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func startProxy(port string, urlString string, cacheSize int) {

	port = ":" + port

	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedUrl)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = parsedUrl.Host
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
