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

func generateNewChannelLink(link, id, ch_id string) string {
	// "auto <channel>" might be replaced with just "<channel>". Testing needed here...
	return `{"js":{"id":"` + ch_id + `","cmd":"auto ` + specialLinkEscape(link) + `","streamer_id":0,"link_id":` + id + `,"load":0,"error":""},"text":"array(6) {\n  [\"id\"]=>\n  string(4) \"` + ch_id + `\"\n  [\"cmd\"]=>\n  string(99) \"` + specialLinkEscape(link) + `\"\n  [\"streamer_id\"]=>\n  int(0)\n  [\"link_id\"]=>\n  int(` + id + `)\n  [\"load\"]=>\n  int(0)\n  [\"error\"]=>\n  string(0) \"\"\n}\ngenerated in: 0.01s; query counter: 8; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`
}

func specialLinkEscape(i string) string {
	return strings.ReplaceAll(i, "/", "\\/")
}
