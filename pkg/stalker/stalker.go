package stalker

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Portal stores details about stalker portal
type Portal struct {
	SerialNumber string
	DeviceID     string
	DeviceID2    string
	Signature    string
	MAC          string
	Username     string
	Password     string
	Location     string
	TimeZone     string
	Token        string
}

var httpClient = &http.Client{
	Timeout: time.Second * 10,
}

// Start connects to stalker portal, authenticates, starts watchdog etc.
func (p *Portal) Start() error {
	// Reserve token in Stalker portal
	if err := p.handshake(); err != nil {
		return err
	}

	// Authorize token (associate with credentials)
	if err := p.authenticate(); err != nil {
		return err
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
	req.Header.Set("X-User-Agent", "Model: MAG254; Link: Ethernet")
	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("Cookie", "PHPSESSID=null; sn="+url.PathEscape(p.SerialNumber)+"; mac="+url.PathEscape(p.MAC)+"; stb_lang=en; timezone="+url.PathEscape(p.TimeZone))

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
	_, err := p.httpRequest(p.Location + "server/load.php?action=get_events&event_active_id=0&init=0&type=watchdog&cur_play_type=1&JsHttpRequest=1-xml")
	if err != nil {
		return err
	}
	return nil
}
