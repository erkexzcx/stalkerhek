package stalker

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (p *Portal) request(link string, maxAttempts int) ([]byte, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (QtEmbedded; U; Linux; C) AppleWebKit/533.3 (KHTML, like Gecko) MAG200 stbapp ver: 4 rev: 2116 Mobile Safari/533.3")
	req.Header.Set("X-User-Agent", "Model: "+p.Model+"; Link: Ethernet")
	req.Header.Set("Authorization", "Bearer "+p.Token)

	cookieText := "PHPSESSID=null; sn=" + url.QueryEscape(p.SerialNumber) + "; mac=" + url.QueryEscape(p.MAC) + "; stb_lang=en; timezone=" + url.QueryEscape(p.TimeZone) + ";"

	req.Header.Set("Cookie", cookieText)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if maxAttempts <= 0 {
			return nil, err
		} else {
			time.Sleep(time.Second)
			return p.request(link, maxAttempts-1)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if maxAttempts <= 0 {
			return nil, errors.New("Site '" + link + "' returned " + resp.Status)
		} else {
			time.Sleep(time.Second)
			return p.request(link, maxAttempts-1)
		}
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if maxAttempts <= 0 {
			return nil, err
		} else {
			time.Sleep(time.Second)
			return p.request(link, maxAttempts-1)
		}
	}

	return contents, nil
}

func specialLinkEscape(i string) string {
	return strings.ReplaceAll(i, "/", "\\/")
}
