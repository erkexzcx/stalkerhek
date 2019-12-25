package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func handleStalkerRequest(w http.ResponseWriter, r *http.Request) {

	q := r.URL.Query()

	_, hasAction := q["action"]

	if r.URL.Path == "/stalker_portal/server/load.php" {
		if hasAction {
			switch q["action"][0] {
			case "logout":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				return
			case "handshake":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputHandshake)
				return
			case "do_auth":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputDoAuth)
				return
			case "get_profile":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputGetProfile)
				return
			case "get_events":
				_, ok := q["type"]
				if ok && q["type"][0] == "watchdog" {
					cacheMutex.RLock()
					w.Write(*(cacheMap["watchdog"]))
					cacheMutex.RUnlock()
					return
				}
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputGetProfile)
				return
			case "get_epg_info":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				cacheMutex.RLock()
				w.Write(*(cacheMap["epg"]))
				cacheMutex.RUnlock()
				return
			}
		}
	}

	// Rewrite any identifying parameter, such as serial number, username, device ID etc...

	_, ok := q["login"]
	if ok {
		q.Set("login", login)
	}
	_, ok = q["password"]
	if ok {
		q.Set("password", password)
	}
	_, ok = q["sn"]
	if ok {
		q.Set("sn", sn)
	}
	_, ok = q["device_id"]
	if ok {
		q.Set("device_id", deviceID)
	}
	_, ok = q["device_id2"]
	if ok {
		q.Set("device_id2", deviceID2)
	}
	_, ok = q["signature"]
	if ok {
		q.Set("signature", signature)
	}
	r.URL.RawQuery = q.Encode()

	requestURI := r.URL.RequestURI()

	if hasAction || !strings.HasSuffix(r.URL.Path, ".php") {
		cacheMutex.RLock()
		content, ok := cacheMap[requestURI]
		cacheMutex.RUnlock()
		if ok {
			w.Write(*content)
			log.Println("Loaded from cache", portalURLDomain+requestURI)
			return
		}
	}

	// Forward request to stalker portal and return it's output and the same HTTP code
	log.Println(portalURLDomain + requestURI)
	resp, err := getRequest(portalURLDomain + requestURI)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// Save cache:
	if hasAction || !strings.HasSuffix(r.URL.Path, ".php") {
		cacheMutex.Lock()
		cacheMap[requestURI] = &body
		cacheMutex.Unlock()
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
