package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)


func main() {

	port := ":8000"
	urlString := "https://www.google.com"
	parsedUrl, err := url.Parse(urlString)

	if err != nil {
		fmt.Println("Error: ", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedUrl)
	cacheManager := NewCacheManager()

	fmt.Println("Server is running on port ", port)
	er := http.ListenAndServe(port, cacheMiddleware(proxy, cacheManager))
	if er != nil {
		fmt.Println("Error: ", er)
	}
}
