package proxy

import (
	"hash/fnv"
	"net/http"
	"net/url"
	"strings"
)

func getRequest(link string, originalRequest *http.Request) (*http.Response, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range originalRequest.Header {
		switch k {
		case "Authorization":
			req.Header.Set("Authorization", "Bearer "+config.Portal.Token)
		case "Cookie":
			cookieText := "PHPSESSID=null; sn=" + url.QueryEscape(config.Portal.SerialNumber) + "; mac=" + url.QueryEscape(config.Portal.MAC) + "; stb_lang=en; timezone=" + url.QueryEscape(config.Portal.TimeZone) + ";"
			req.Header.Set("Cookie", cookieText)
		default:
			req.Header.Set(k, v[0])
		}
	}

	return http.DefaultClient.Do(req)
}

func addHeaders(from, to http.Header) {
	for k, v := range from {
		to.Set(k, strings.Join(v, "; "))
	}
}

func specialLinkEscape(i string) string {
	return strings.ReplaceAll(i, "/", "\\/")
}

func stringToHash(s string) string {
	hash := fnv.New64a()
	hash.Write([]byte(s))
	return string(hash.Sum64())
}
