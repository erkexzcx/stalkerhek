# WIP - Stalkerhek

*Stalker* is a pretty popular IPTV streaming solution. Usually you can buy a TV box with preconfigured credentials and stalker portal (URL). Stalker TV box has it's own unique device ID (actually 2 IDs), signature, mac address and so on. On top of that, if you share your authentication details and set-up another TV box, the other one will get disconnected, making it possible to only watch on a single device at the same time.

This software allows you to watch Stalker TV on VLC or Kodi and on multiple devices. It serves IPTV as M3U playlist and acts as a proxy.

# Advantages:

Here are some advatages:
* Play on VLC rather than using emulator or STB boxes.
* This app is the only way to play Stalker IPTV on jailbroken PS3, Movian player.

# How to use

This app might contain bugs and "it works for me", so you have been warned.

## 1. Extract stalker credentials and other stuff required to connect

To extract all the authentication details, use wireshark to capture HTTP requests and analyse them by hand. I used **capture** filter `port 80 and tcp[((tcp[12:1] & 0xf0) >> 2):4] = 0x47455420` and **display** filter `http.request.method == GET`. You will likely want to use MITM attack using [arpspoof](https://www.irongeek.com/i.php?page=security/arpspoof). You will also need to restart TV box when capturing requests to see your TV box logging into stalker portal with stored authentication details. If you are smart/lucky enough, you can use port mirroring on your router and wireshark on the mirrored-to port. Anyway, you must capture the traffic in any way you can.

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

All this info will be visible in the URLs or request headers (everything should exist in wireshark capture). Let's hope SSL is not in use, otherwise I can't advise how to decrypt such traffic.

## 2. Append extracted details to config file

```
cp stalkerhek.example.yml stalkerhek.yml
vim stalkerhek.yml
```

## 3. Build & start application

First, you have to download & install Golang from [here](https://golang.org/doc/install). DO NOT install Golang from the official repositories because they contain outdated version which is not working with this project.

To ensure Golang is installed successfully, test it with `go version` command. Example:
```
$ go version
go version go1.16.2 linux/amd64
```

Then build the application and test it:
```
go build -ldflags="-s -w" -o "stalkerhek" ./cmd/stalkerhek/main.go
./stalkerhek -help
./stalkerhek -config stalkerhek.yml -bind 0.0.0.0:9999
```

P.S. If you are getting some sort of strange 'A' error - try starting again.

## 4. Use application

I suggest first testing with CURL:
```
curl http://<ipaddr>:8987/iptv
```

You might see that there are no channels - in such case simply restart this application and try again.

If there are channels loaded, use above URL in VLC/Kodi. :)
