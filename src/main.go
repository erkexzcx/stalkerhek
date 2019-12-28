package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const configFilePath = "config.yaml"

type config struct {
	SerialNumber string `yaml:"serial_number"`
	DeviceID     string `yaml:"device_id"`
	DeviceID2    string `yaml:"device_id2"`
	Signature    string `yaml:"signature"`
	MAC          string `yaml:"mac"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Portal       string `yaml:"portal_url"`
	TimeZone     string `yaml:"time_zone"`
	Token        string `yaml:"token"`
}

var conf config // Mutex not needed, since settings are no longer edited after loading them

func main() {

	// Before starting up...
	loadConfig()
	validateConfig()
	pathEscapeConfig()

	log.Println("Starting...")

	authenticate()

	// Keep-alive with the stalker portal
	go func() {
		for {
			watchdogUpdate()
			time.Sleep(2 * time.Minute)
		}
	}()

	generatePlaylist()

	// Constantly clear old media cache
	go func() {
		for {
			removeOldCache()
			time.Sleep(5 * time.Second)
		}
	}()

	http.HandleFunc("/iptv", handlePlaylistRequest)
	http.HandleFunc("/iptv/", handleRequest)

	log.Println("Started!")
	log.Fatal(http.ListenAndServe(":8987", nil))
}

var httpClient = &http.Client{}

func getRequest(link string) (*http.Response, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (QtEmbedded; U; Linux; C) AppleWebKit/533.3 (KHTML, like Gecko) MAG200 stbapp ver: 4 rev: 2116 Mobile Safari/533.3")
	req.Header.Set("X-User-Agent", "Model: MAG254; Link: Ethernet")
	req.Header.Set("Authorization", "Bearer "+conf.Token)
	req.Header.Set("Cookie", "PHPSESSID=null; sn="+conf.SerialNumber+"; mac="+conf.MAC+"; stb_lang=en; timezone="+conf.TimeZone)

	return httpClient.Do(req)
}

func authenticate() {
	type tokenStruct struct {
		Js struct {
			Token string `json:"token"`
		} `json:"js"`
	}
	var tmpToken tokenStruct

	resp, err := getRequest(conf.Portal + "server/load.php?type=stb&action=handshake&prehash=0&token=" + conf.Token + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Fatalln(err)
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(contents, &tmpToken)
	if tmpToken.Js.Token != "" {
		conf.Token = tmpToken.Js.Token
		log.Println("Using server's issued token", conf.Token)
	} else {
		log.Println("Using user's provided token", conf.Token)
	}
	resp.Body.Close()

	// Since we have a token, we need to authorize it (associate it with your credentials)
	type resStruct struct {
		Js bool `json:"js"`
	}
	var tmpRes resStruct
	resp, err = getRequest(conf.Portal + "server/load.php?type=stb&action=do_auth&login=" + conf.Username + "&password=" + conf.Password + "&device_id=" + conf.DeviceID + "&device_id2=" + conf.DeviceID2 + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	contents, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(contents, &tmpRes)
	if tmpRes.Js {
		log.Println("Authenticated successfully")
	} else {
		log.Fatalln("Failed to authenticate token")
	}
}

func watchdogUpdate() {
	req, err := getRequest(conf.Portal + "server/load.php?action=get_events&event_active_id=0&init=0&type=watchdog&cur_play_type=1&JsHttpRequest=1-xml")
	if err != nil {
		log.Println(err)
	}
	req.Body.Close()
}

func loadConfig() {
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln(err)
	}

	err = yaml.Unmarshal(content, &conf)
	if err != nil {
		log.Fatalln(err)
	}
}

func validateConfig() {
	/* Some very basic checks... */
	if strings.Replace(conf.SerialNumber, " ", "", 1) != conf.SerialNumber {
		log.Fatalln("serial number cannot contain spaces")
	}
	if conf.SerialNumber == "" {
		log.Fatalln("serial number cannot be empty")
	}

	if strings.Replace(conf.DeviceID, " ", "", 1) != conf.DeviceID {
		log.Fatalln("device ID cannot contain spaces")
	}
	if conf.DeviceID == "" {
		log.Fatalln("device ID cannot be empty")
	}

	if strings.Replace(conf.DeviceID2, " ", "", 1) != conf.DeviceID2 {
		log.Fatalln("device ID2 cannot contain spaces")
	}
	if conf.DeviceID2 == "" {
		log.Fatalln("device ID2 cannot be empty")
	}

	if strings.Replace(conf.Signature, " ", "", 1) != conf.Signature {
		log.Fatalln("signature cannot contain spaces")
	}
	if conf.Signature == "" {
		log.Fatalln("signature cannot be empty")
	}

	if strings.Replace(conf.MAC, " ", "", 1) != conf.MAC {
		log.Fatalln("MAC cannot contain spaces")
	}
	if conf.MAC == "" {
		log.Fatalln("MAC cannot be empty")
	}

	if strings.Replace(conf.Username, " ", "", 1) != conf.Username {
		log.Fatalln("username cannot contain spaces")
	}
	if conf.Username == "" {
		log.Fatalln("username cannot be empty")
	}

	if conf.Password == "" {
		log.Fatalln("password cannot be empty")
	}

	if !strings.HasSuffix(conf.Portal, "/stalker_portal/") {
		log.Fatalln("invalid Stalker portal: it must end with '/stalker_portal/'")
	}

	if strings.Replace(conf.TimeZone, " ", "", 1) != conf.TimeZone {
		log.Fatalln("timezone cannot contain spaces")
	}
	if conf.TimeZone == "" {
		log.Fatalln("timezone cannot be empty")
	}

	if strings.Replace(conf.Token, " ", "", 1) != conf.Token {
		log.Fatalln("token cannot contain spaces")
	}
	if conf.Token == "" {
		log.Fatalln("token cannot be empty")
	}
}

func pathEscapeConfig() {
	// Everything except portal URL
	conf.DeviceID = url.PathEscape(conf.DeviceID)
	conf.DeviceID2 = url.PathEscape(conf.DeviceID2)
	conf.MAC = url.PathEscape(conf.MAC)
	conf.Password = url.PathEscape(conf.Password)
	conf.SerialNumber = url.PathEscape(conf.SerialNumber)
	conf.Signature = url.PathEscape(conf.Signature)
	conf.TimeZone = url.PathEscape(conf.TimeZone)
	conf.Token = url.PathEscape(conf.Token)
	conf.Username = url.PathEscape(conf.Username)
}

func generatePlaylist() {

	type cstruct struct {
		Js struct {
			Data []struct {
				Name string `json:"name"`
				Cmd  string `json:"cmd"`
				Logo string `json:"logo"`
			} `json:"data"`
		} `json:"js"`
	}
	var cs cstruct

	content, err := ioutil.ReadFile("/tmp/channelsCache")
	if err != nil {
		panic(err)
	}

	// req, err := getRequest(conf.Portal + "server/load.php?type=itv&action=get_all_channels&force_ch_link_check=&JsHttpRequest=1-xml")
	// if err != nil {
	// 	panic(err)
	// }
	// defer req.Body.Close()
	// content, err := ioutil.ReadAll(req.Body)
	// if err != nil {
	// 	panic(err)
	// }

	// err = ioutil.WriteFile("/tmp/channelsCache", content, 0644)
	// if err != nil {
	// 	panic(err)
	// }

	if err := json.Unmarshal(content, &cs); err != nil {
		panic(err)
	}

	for _, v := range cs.Js.Data {
		tvchannelsMap[v.Name] = &tvchannel{
			Cmd:  v.Cmd,
			Logo: v.Logo,
		}
	}
}
