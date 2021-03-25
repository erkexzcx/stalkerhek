# Stalkerhek

[![Build Status](https://travis-ci.com/erkexzcx/stalkerhek.svg?branch=master)](https://travis-ci.com/erkexzcx/stalkerhek) 
[![Go Report Card](https://goreportcard.com/badge/github.com/erkexzcx/stalkerhek)](https://goreportcard.com/report/github.com/erkexzcx/stalkerhek)

*Stalker* is a popular IPTV streaming solution. You can buy a preconfigured TV box or just a Stalker account which can be used in special TV Boxes or emulators such as [Stbemu](https://play.google.com/store/search?q=StbEmu). Stalker account consists of portal (URL), username/password (optional), 2 unique device IDs, signature, mac address and so on. On top of that, if you setup Stalker account in another TV Box, the other one will get disconnected, making it possible to only watch on a single device at the same time.

**Stalkerhek** application allows you to share single Stalker account between multiple STB boxes as well as makes it possible to watch Stalker IPTV using simple video players, such as VLC.

Advantages:
* Watch Stalker IPTV on a simple media players (e.g. VLC).
* Watch on multiple devices, even from different source IP addresses, at the same time.

Disadvantages/missing features:
* Based on reverse-engineering. Expect some channels/configurations not to work at all.
* No caching (if 5 viewers are watching the same IPTV channel at the same time, then IPTV channel will receive 5x more requests).
* No VOD.
* No EPG.

# Services

There 2 different services provided by Stalkerhek. They both can be used at the same time.

## HLS service

This service spawns a proxy server which converter from Stalker IPTV to HLS format, allowing to watch Stalker IPTV using simple video players, such as VLC. This service serves channels list as HLS (M3U) playlist and rewrites all further metadata/media links, forcing all the further traffic to go through this application and effectively hide original viewer's source IP from Stalker middleware.

## Proxy service

This service spawns a proxy server which is intended to be used for single Stalker account sharing between different STB boxes. Speaking about internals - STB boxes configured to use this service as Stalker portal will always be able work, because this service silently ignores authentication, watchdog and logoff requests (returns expected, but fake replies), while full functionality of real Stalker portal will be accessible.

# Usage

## 1. Extract Stalker authorization details from STB box

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

## 2. Configuration

Create configuration file as per below commands:

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
./stalkerhek -config stalkerhek.yml
```

If you decide to edit the code, you can quickly test if it works without compiling it:
```bash
go run ./cmd/stalkerhek/main.go -help
go run ./cmd/stalkerhek/main.go -config stalkerhek.yml
```

## 4. Usage

### HLS service

I suggest first testing with CURL:
```bash
curl http://<ipaddr>:9999/iptv
```

If you see there are channels loaded, use above URL in VLC.

### Proxy service

Check if you can get response using CURL from the real Stalker middleware URL:
```
curl http://example.com/stalker_portal/server/load.php
```

Do the same, but replace host:port with this service host:port as per below example:
```
curl http://ipaddr>:8888/stalker_portal/server/load.php
```

You should get the same response.

If response was the same, it means proxy service is working and you can now use this proxy service URL as stalker portal URL.

Note that this service is **not appending**, but **replacing** values on-the-fly. It means you have to add any fake credentials, serial numbers, device IDs etc. to your Stalker client in order for it to work.

**Instructions for Kodi**: In Kodi Stalker addon settings, use Portal URL in the same format as you tried above (`http://ipaddr>:8888/stalker_portal/server/load.php`). Add any fake username/password, any numbers/letters in device IDs, serial numbers etc. Restart Kodi and :tada:.

## 5. Installation guidelines

1. Copy/paste file `stalkerhek.service` to `/etc/systemd/system/stalkerhek.service`.
2. Edit `/etc/systemd/system/stalkerhek.service` file and replace `myuser` with your non-root user. Also change paths if necessary.
3. Perform `systemctl daemon-reload`.
4. Use `systemctl <enable/disable/start/stop> stalkerhek.service` to manage this service.

P.S. Sorry for those who are looking for binary releases or dockerfile - I will consider it when this project becomes more stable.
