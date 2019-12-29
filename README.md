# WIP - Stalker-hek

*Stalker* is a pretty popular IPTV streaming solution. Usually you can buy a TV box with preconfigured credentials and stalker portal (URL). Stalker TV box has it's own unique device ID (actually 2 IDs), signature, mac address and so on. On top of that, if you share your authentication details and set-up another TV box, the other one will get disconnected, making it possible to only watch on a single device at the same time.

This software allows you to watch Stalker TV on VLC or Kodi and on multiple devices. It serves IPTV as M3U playlist and acts as a proxy.

Things that do not work:
* application/octet-stream is not working. VLC/Kodi is just not loading it...
* Recordings that are served together with channels from stalker portal (from my stalker box)
* Missing categories. Will add later
* No EPG and not going to be any time soon

# How to use

This app is incomplete, contains bugs and "it works for me", so you have been warned.

## 1. Extract stalker credentials (and other required stuff)

To extract all the authentication details, use wireshark to capture HTTP requests and analyse them by hand. I used **capture** filter `port 80 and tcp[((tcp[12:1] & 0xf0) >> 2):4] = 0x47455420` and **display** filter `http.request.method == GET`. You will likely want to use MITM attack using [arpspoof](https://www.irongeek.com/i.php?page=security/arpspoof). You will also need to restart TV box when capturing requests to see your TV box logging into stalker portal with stored authentication details.

You will need the following details extracted from the wireshark logs:
* sn
* device_id
* device_id2
* signature
* mac
* login
* password
* timezone
* address of the stalker middleware server. If your tv box connets to `http://domain.example.com/stalker_portal/c/...` then it's going to be `http://domain.example.com/stalker_portal/`.

All this info will be visible in the URLs or Cookies (wireshark will capture everything).

## 2. Append extracted details to config file

```
mv config.example.yaml config.yaml
vim config.yaml
```

## 3. Start application

You will also need Go programming language installed:
```
go run main.go
```
Note that this will also stops your existing IPTV box from working. Reboot it so it connects and works again.

## 4. Use VLC

Use VLC, Kodi or test if link is working in browser:
```
vlc http://<ipaddr>:/8987/iptv
```
