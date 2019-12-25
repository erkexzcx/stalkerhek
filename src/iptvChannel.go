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
	Cmd                string
	Logo               string
	Link               string
	LinkRoot           string
	LinkCache          []byte
	LinkCacheCreation  time.Time
	LastLinkRootAccess time.Time
	Mux                sync.RWMutex
}

func (channel *tvchannel) LinkRootStillValid() bool {
	if channel.LastLinkRootAccess.IsZero() || time.Since(channel.LastLinkRootAccess).Seconds() > 15 {
		return false
	}
	return true
}

func (channel *tvchannel) LinkCacheValid() bool {
	if channel.LinkCacheCreation.IsZero() || time.Since(channel.LinkCacheCreation).Seconds() > 5 {
		return false
	}
	return true
}

func (channel *tvchannel) RefreshLink() {
	type tmpstruct struct {
		Js struct {
			Cmd string `json:"cmd"`
		} `json:"js"`
	}
	var tmp tmpstruct

	// Mutex is not needed, since parent codeblock is already RLocked
	resp, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?action=create_link&type=itv&cmd=" + url.PathEscape(channel.Cmd) + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Println("Failed to resolve channel")
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(content, &tmp); err != nil {
		panic(err)
	}

	strs := strings.Split(tmp.Js.Cmd, " ")
	if len(strs) == 2 {
		channel.Link = strs[1]
		channel.LinkRoot = deleteAfterLastSlash(strs[1])
	}
	return
}
