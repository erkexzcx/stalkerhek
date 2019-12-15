# WIP - Stalker-hek

Every stalker account can only be used with single TV box, which means you can't share your Stalker IPTV account with friends or use it on several other devices.

This application allows you to use your stalker IPTV on unlimited amount of devices, by using your single stalker account (yup, multiple TVs, sharing with friends etc).

# How it works

When this application is started, it authenticates with stalker portal using your defined account and defined portal URL. And the rest is just forwarding requests to your defined Stalker middleware, while ignoring authentication requests.

This is how your typical set-tup box works:
```
[TV box] <--> [stalker portal]
```

And this is how it would work with this app:
```
[TV box] <--> [this app] <--> [stalker portal]
[TV box] <--> 
[TV box] <--> 
[TV box] <--> 
```

# How to use

This app is not finished at all, might contain bugs and "it works for me", so you have been warned.

## 1. Extract stalker credentials (and other required stuff)

Use wireshark. I used **capture** filter `port 80 and tcp[((tcp[12:1] & 0xf0) >> 2):4] = 0x47455420` and **display** filter `http.request.method == GET`. In case you want to spawn MITM attack, see [this](https://www.irongeek.com/i.php?page=security/arpspoof). You will need to restart IPTV box when captuing requests to see your IPTV box logging into stalker portal with stored credentials and other stuff.

You will need the following details extracted from the wireshark logs:
* sn
* device_id
* device_id2
* signature
* mac
* login
* password
* timezone
* address of the stalker middleware server. If your tv box connets to `http://domain.example.com/stalker_portal/c/...` then it's going to be `http://domain.example.com`.

All this info will be visible in the URLs or Cookies (wireshark will capture everything).

## 2. Hardcode extracted authentication details

Edit `main.go` file and update constants `const` with details you've extracted using Wireshark. I left dummy values for you, so it's easier to understand what is needed.

## 3. Start application

You will also need Go programming language installed:
```
go run main.go
```
Note that this will also stops your existing IPTV box from working. Reboot it so it connects and works again.

## 4. Use KODI

Install Kodi and install Stalker TV addon. Configure this addon and set the following details:

* Server Address: `http://<ipaddr>/stalker_portal/c/` (you MUST enter full URL as shown. Use IP address of the device where this app is running.
* Login - <anything>
* Password - <anything>
* token - <anything>
* Serial Number - <anything>
* Device ID - <anything>
* Device ID2 - <anything>
* Signature - <anything>
  
<anything> literally means anything. Like `asdfasdf`. :)
  
Restart Kodi - your IPTV should work in Kodi.

# Other notes

* I don't own any Stalker IPTV boxes. Used friend's box on weekend, so no active development. Maybe in the future. :)
* Yes, this is Ugly Golang code. It was reverse engineering anyway, so I already rewrote it serveral times until got it to work.
