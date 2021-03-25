# Stalkerhek

# DO NOT USE THIS GIT BRANCH. IT'S FOR DEVELOPMENT PURPOSES!!!

[![Build Status](https://travis-ci.com/erkexzcx/stalkerhek.svg?branch=master)](https://travis-ci.com/erkexzcx/stalkerhek) 
[![Go Report Card](https://goreportcard.com/badge/github.com/erkexzcx/stalkerhek)](https://goreportcard.com/report/github.com/erkexzcx/stalkerhek)

*Stalker* is a popular IPTV streaming solution. You can buy a preconfigured TV box or just a Stalker account which can be used in special TV Boxes or emulators such as [Stbemu](https://play.google.com/store/search?q=StbEmu). Stalker account consists of portal (URL), username/password (optional), 2 unique device IDs, signature, mac address and so on. On top of that, if you setup Stalker account in another TV Box, the other one will get disconnected, making it possible to only watch on a single device at the same time.

**Stalkerhek** is a proxy server and converter from Stalker IPTV to HLS format, allowing to watch Stalker IPTV using simple video players, such as VLC. Stalkerhek serves Stalker's provided channels list as HLS (M3U) playlist and rewrites all further links, forcing all IPTV requests to go through this application and effectively hiding original viewer's source IP from Stalker middleware.

Advantages:
* Watch Stalker IPTV on a simple media players (e.g. VLC).
* Watch on multiple devices, even from different source IP addresses, at the same time.

Disadvantages/missing features:
* Based on reverse-engineering. Expect some channels/configurations not to work at all.
* No caching (if 5 viewers are watching the same IPTV channel at the same time, then IPTV channel will receive 5x more requests).
* No VOD.
* No EPG.

# Usage

## 1. Extract Stalker authorization details from TV box

You can skip this step if you have below details already **and** you are sure that they are working.

To extract all the authentication details, use wireshark to capture HTTP requests and analyse them by hand. I used **capture** filter `port 80 and tcp[((tcp[12:1] & 0xf0) >> 2):4] = 0x47455420` and **display** filter `http.request.method == GET`. You will likely want to use MITM attack using [arpspoof](https://www.irongeek.com/i.php?page=security/arpspoof). You will also need to restart TV box when capturing requests to see your TV box logging into stalker portal with stored authentication details. If you are smart/lucky enough, you can use port mirroring on your router and wireshark on the mirrored-to port. Anyway, you must capture the traffic in any way you can.

**Tip**: In wireshark you need to find HTTP request containing `action=get_profile` which contains most of the details. For username/password pair, you should search for URL containing `action=do_auth`, which might *not* exist if you do not require credentials for authentication with Stalker middleware. All of this can be filtered out using single display filter `http.request.full_uri contains "action=do_auth" or http.request.full_uri contains "action=get_profile"`.

You will need the following details extracted from the wireshark logs (see `stalkerhek.example.yml` file):
* model - from request headers
* sn (serial number) - from URL
* device_id - from URL
* device_id2 - from URL
* signature - from URL
* mac - from request headers
* login - from URL
* password - from URL
* timezone - from request headers
* location (URL address) - from URL
* token - from request headers, next to "Bearer ". Does not matter that much since stalker server should issue new token if provided is in use.

Regarding URL address/location: If your tv box connets to `http://domain.example.com/stalker_portal/server/load.php?...` then it's going to be `http://domain.example.com/stalker_portal/server/load.php`. If it connects to `http://domain.example.com/portal.php?...`, then it's going to be `http://domain.example.com/portal.php`. Wireshark will tell you where it connects. :)

All this info will be visible in the URLs or request headers (everything should exist in wireshark capture).

## 2. Create configuration file

```bash
cp stalkerhek.example.yml stalkerhek.yml
vim stalkerhek.yml
```

## 3. Build application

First, you have to download & install Golang from [here](https://golang.org/doc/install). DO NOT install Golang from the official repositories because they contain outdated version which is not working with this project.

To ensure Golang is installed successfully, test it with `go version` command. Example:
```bash
$ go version
go version go1.16.2 linux/amd64
```

Then build the application and test it:
```bash
go build -ldflags="-s -w" -o "stalkerhek" ./cmd/stalkerhek/main.go
./stalkerhek -help
./stalkerhek -config stalkerhek.yml -bind 0.0.0.0:9999
```

If you decide to edit the code, you can quickly test if it works without compiling it:
```bash
go run ./cmd/stalkerhek/main.go -help
go run ./cmd/stalkerhek/main.go -bind 0.0.0.0:9999
```

## 4. Run application

I suggest first testing with CURL:
```bash
curl http://<ipaddr>:8987/iptv
```

You might see that there are no channels - in such case simply restart this application and try again.

If there are channels loaded, you can use above URL in VLC/Kodi. :)

## 5. Installation guidelines

1. Copy/paste file `stalkerhek.service` to `/etc/systemd/system/stalkerhek.service`.
2. Edit `/etc/systemd/system/stalkerhek.service` file and replace `myuser` with your non-root user. Also change paths if necessary.
3. Perform `systemctl daemon-reload`.
4. Use `systemctl <enable/disable/start/stop> stalkerhek.service` to manage this service.

P.S. Sorry for those who are looking for binary releases or dockerfile - I will consider it when this project becomes more stable.
