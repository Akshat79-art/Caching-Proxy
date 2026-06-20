package cmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)


func startProxy(port string, urlString string) {

	port = ":" + port
	
	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedUrl)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = parsedUrl.Host
	}

	cacheManager := NewCacheManager()

	fmt.Println("Server is running on port ", port)
	er := http.ListenAndServe(port, cacheMiddleware(proxy, cacheManager))
	if er != nil {
		fmt.Println("Error: ", er)
	}
}