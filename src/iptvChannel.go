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
	LinkAccessTime    time.Time
	LinkRoot          string
	SessionUpdateTime time.Time
	Mux               sync.RWMutex

	// Store link's cache here
	Cache            []byte
	CacheContentType string
	CacheMux         sync.Mutex
}

func (c *tvchannel) LinkCacheValid() bool {
	c.Mux.RLock()
	defer c.Mux.RUnlock()
	if c.LinkAccessTime.IsZero() || time.Since(c.LinkAccessTime).Seconds() > 3 || len(c.Cache) == 0 {
		return false
	}
	return true
}

func (c *tvchannel) SessionValid() bool {
	c.Mux.RLock()
	defer c.Mux.RUnlock()
	if c.SessionUpdateTime.IsZero() || time.Since(c.SessionUpdateTime).Seconds() > 20 {
		return false
	}
	return true
}

func (c *tvchannel) RefreshLink() {
	// Create TV channel API link
	c.Mux.RLock()
	link := conf.Portal + "server/load.php?action=create_link&type=itv&cmd=" + url.PathEscape(c.Cmd) + "&JsHttpRequest=1-xml"
	c.Mux.RUnlock()

	// Query that API link and download content
	resp, err := getRequestAPI(link)
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
		c.LinkAccessTime = time.Now()
		c.SessionUpdateTime = time.Now()
		c.Mux.Unlock()
	} else {
		log.Println("Failed to extract TV channel URL from Stalker API...")
	}
}

func deleteAfterLastSlash(str string) string {
	return str[0 : strings.LastIndex(str, "/")+1]
}
