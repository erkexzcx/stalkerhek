package stalker

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

var httpClient = &http.Client{
	Timeout: time.Second * 10,
}

// Start connects to stalker portal, authenticates, starts watchdog etc.
func (p *Portal) Start() error {
	// Reserve token in Stalker portal
	if err := p.handshake(); err != nil {
		return err
	}

	// Authorize token if credentials are given
	if p.Username != "" && p.Password != "" {
		if err := p.authenticate(); err != nil {
			return err
		}
	}

	// Run watchdog function once to check for errors:
	if err := p.watchdogUpdate(); err != nil {
		return err
	}

	// Run watchdog function every 2 minutes:
	go func() {
		for {
			time.Sleep(2 * time.Minute)
			if err := p.watchdogUpdate(); err != nil {
				log.Fatalln(err)
			}
		}
	}()

	return nil
}

func (p *Portal) httpRequest(link string) ([]byte, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (QtEmbedded; U; Linux; C) AppleWebKit/533.3 (KHTML, like Gecko) MAG200 stbapp ver: 4 rev: 2116 Mobile Safari/533.3")
	req.Header.Set("X-User-Agent", "Model: "+p.Model+"; Link: Ethernet")
	req.Header.Set("Authorization", "Bearer "+p.Token)

	cookieText := "PHPSESSID=null; mac=" + url.QueryEscape(p.MAC) + "; stb_lang=en; timezone=" + url.QueryEscape(p.TimeZone) + ";"
	if p.SerialNumber != "" {
		cookieText += " sn=" + url.QueryEscape(p.SerialNumber) + ";"
	}

	req.Header.Set("Cookie", cookieText)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("Site '" + link + "' returned " + resp.Status)
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

// WatchdogUpdate performs watchdog update request.
func (p *Portal) watchdogUpdate() error {
	_, err := p.httpRequest(p.Location + "?action=get_events&event_active_id=0&init=0&type=watchdog&cur_play_type=1&JsHttpRequest=1-xml")
	if err != nil {
		return err
	}
	return nil
}
