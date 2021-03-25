package proxy

import (
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
			req.Header.Set("Authorization", "Bearer "+portal.Token)
		case "Cookie":
			cookieText := "PHPSESSID=null; sn=" + url.QueryEscape(portal.SerialNumber) + "; mac=" + url.QueryEscape(portal.MAC) + "; stb_lang=en; timezone=" + url.QueryEscape(portal.TimeZone) + ";"
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
