# Stalkerhek

[![Build Status](https://travis-ci.com/erkexzcx/stalkerhek.svg?branch=master)](https://travis-ci.com/erkexzcx/stalkerhek) 
[![Go Report Card](https://goreportcard.com/badge/github.com/erkexzcx/stalkerhek)](https://goreportcard.com/report/github.com/erkexzcx/stalkerhek)

*Stalker* (usually referred as "Stalker middleware") is a popular IPTV streaming solution. It is originally intended to be used with a special TV boxes (also known as "STB" or "STB boxes"), but software solutions can also be used (e.g. [Stbemu](https://play.google.com/store/search?q=StbEmu)). Benefits of Stalker middleware include user/TV box authorization, IPTV, integrated EPG, VOD, media library and language support, making it nice all-in-one solution for a TV box. Underlaying IPTV video streams are being provided in HLS which is widely supported format by almost all video players (e.g. VLC).

STB boxes usually have hardcoded parameters, such as MAC address, 2 device IDs and signature that cannot be changed. Providers themselves can setup such STB boxes by enterring Stalker portal URL with credentials, then mapping device's hardcoded parameters to the added credentials, making it (theoretically) impossible to clone STB box. STB boxes will also contact Stalker middleware saying "hey, my username is X and my token is Y, please log off all other devices under my username that use any other token" which also makes it possible to only use a single STB box at a time.

Looking from security perspective, STB boxes communicate with Stalker middleware in simple HTTP requests (without SSL) and sends its hardcoded parameters/credentials in both URL query and request headers (think of `http://example.com/load.php?username=abc&password=aaa`) in a plain text.

**Stalkerhek** is a Stalker middleware proxy application that allows you to share the same Stalker portal account on (theoretically) unlimited amount of STB boxes and makes it possible to watch Stalker portal IPTV in simple video players, such as VLC. This application itself connects to Stalker portal, authenticates with it and keeps sending keep-alive requests to remain connected. The rest is being done by this application's [#Services](#Services).

# Services

There are 2 services provided by Stalkerhek. They both can be used at the same time.

## HLS service

*Used for viewing Stalker IPTV in simple video players, such as VLC.*

This service spawns a HTTP web server that returns HLS playlist of all Stalker IPTV channels when requested. All returned IPTV links are rewritten in a way that all the stream traffic will go through this application, eventually hiding original viewer's source IP from IPTV provider as well as hiding IPTV provider's server IP/host from IPTV viewer.

Note that there is no caching. if 5 devices are watching the same channel, the IPTV provider will receive 5x more requests.

## Proxy service

*Used for sharing single Staler portal credentials between multiple STB boxes. It can also be used for centralized control of STB boxes.*

This service spawns a HTTP web server which is used as a Stalker portal in STB boxes. It forwards all the incoming requests from STB boxes to the real Stalker portal, but on-the-fly rewrites all the credentials and hardcoded parameters.

How your current setup looks like:
```
[STB MAC:A] <--> [Stalker middleware]
```

How it would look if you use this service (it rewrites from MAC address `A` to expected address `B`):
```
[STB MAC:A] <--> [Stalkerhek MAC:B] <--> [Stalker middleware]
```

You are not limited to a single STB box anymore:
```
[STB MAC:A] <-->
[STB MAC:B] <--> [Stalkerhek MAC:B] <--> [Stalker middleware]
[STB MAC:C] <-->
```

**Note** this service will only proxy Stalker middleware communication requests (e.g. retrieving channels list), but not the actual video streams. Provider's servers that are serving the media will be accessed directly, exposing original viewer's source IP address as well as provider's server address.

# Usage

## 1. Extract Stalker authorization details from STB box

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

## 4. Using application

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

Note that this service is **not appending**, but **replacing** values on-the-fly. It means you have to leave credentials, serial numbers, device IDs etc. non empty, otherwise client would not send them and configuration would not work. In other words, add any fake details to your Stalker IPTV client's configuration.

**Instructions for Kodi**: In Kodi Stalker addon settings, use Portal URL in the same format as you tested with CURL above (`http://ipaddr>:8888/stalker_portal/server/load.php`). Add any fake username/password, any numbers/letters in device IDs, serial numbers etc. Restart Kodi and :tada:.

**Instructions for other apps**: Same as Kodi, except Stalker URL is slightly different - `http://ipaddr>:8888/stalker_portal/c`. Add any fake username/password, any numbers/letters in device IDs, serial numbers etc. Restart and :tada:.

## 5. Installation guidelines

1. Copy/paste file `stalkerhek.service` to `/etc/systemd/system/stalkerhek.service`.
2. Edit `/etc/systemd/system/stalkerhek.service` file and replace `myuser` with your non-root user. Also change paths if necessary.
3. Perform `systemctl daemon-reload`.
4. Use `systemctl <enable/disable/start/stop> stalkerhek.service` to manage this service.

P.S. Sorry for those who are looking for binary releases or dockerfile - I will consider it when this project becomes more stable.
