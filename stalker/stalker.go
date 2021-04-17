package stalker

import (
	"log"
	"time"
)

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

// WatchdogUpdate performs watchdog update request.
func (p *Portal) watchdogUpdate() error {
	_, err := p.request(p.Location+"?action=get_events&event_active_id=0&init=0&type=watchdog&cur_play_type=1&JsHttpRequest=1-xml", 3)
	if err != nil {
		return err
	}
	return nil
}
