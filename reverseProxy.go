package main

import (
	"net/http/httputil"
	"net/url"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
}

func reverseProxy() []*Backend {
	rawURLs := []string{
		"http://localhost:9001",
		"http://localhost:9002",
		"http://localhost:9003",
	}
	var backends []*Backend

	for _, raw := range rawURLs {
		u, _ := url.Parse(raw)

		backends = append(backends, &Backend{
			URL:   u,
			Proxy: httputil.NewSingleHostReverseProxy(u),
		})
	}

	return backends

	// fmt.Println(len(backends))
	// fmt.Println(backends[0].URL)
	// fmt.Println(backends[0].URL.Host)

	// for i := 0; i <= 10; i++ {

	// 	fmt.Println(nextBackend(backends).URL.Host)
	// }

}
