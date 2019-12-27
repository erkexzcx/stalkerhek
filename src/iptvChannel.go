package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"
)

type tvchannel struct {
	Cmd               string
	Logo              string
	Link              string
	LastLinkAccess    time.Time
	Mux               sync.RWMutex
	Cache             cache
	CacheCreationTime time.Time
	CacheMux          sync.Mutex

	LinkRoot      string
	LinksCache    map[string]*cache
	LinksCacheMux sync.RWMutex
}

type cache struct {
	Content     []byte
	ContentType string
}

func (c *tvchannel) LinkStillValid() bool {
	c.Mux.RLock()
	defer c.Mux.RUnlock()
	if c.LastLinkAccess.IsZero() || time.Since(c.LastLinkAccess).Seconds() > 15 {
		return false
	}
	return true
}

func (c *tvchannel) LinkCacheValid() bool {
	c.Mux.RLock()
	defer c.Mux.RUnlock()
	if c.CacheCreationTime.IsZero() || time.Since(c.CacheCreationTime).Seconds() > 3 {
		return false
	}
	return true
}

func (c *tvchannel) RefreshLink() {
	// Create TV channel API link
	c.Mux.RLock()
	link := portalURLDomain + "/stalker_portal/server/load.php?action=create_link&type=itv&cmd=" + url.PathEscape(c.Cmd) + "&JsHttpRequest=1-xml"
	c.Mux.RUnlock()

	// Query that API link and download content
	resp, err := getRequest(link)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	// Extract channel URL from content
	type tmpstruct struct {
		Js struct {
			Cmd string `json:"cmd"`
		} `json:"js"`
	}
	var tmp tmpstruct
	if err := json.Unmarshal(content, &tmp); err != nil {
		panic(err)
	}
	strs := strings.Split(tmp.Js.Cmd, " ")
	if len(strs) == 2 {
		c.Mux.Lock()
		c.Link = strs[1]
		c.LinkRoot = deleteAfterLastSlash(strs[1])
		c.LastLinkAccess = time.Now()
		c.Mux.Unlock()
	} else {
		log.Println("Failed to extract TV channel URL from Stalker API...")
	}
}
