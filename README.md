# WIP - Stalker-hek

*Stalker* is a pretty popular IPTV streaming solution. Usually you can buy a TV box with preconfigured credentials and stalker portal (URL). Stalker TV box has it's own unique device ID (actually 2 IDs), signature, mac address and so on. On top of that, if you share your authentication details and set-up another TV box, the other one will get disconnected, making it possible to only watch on a single device at the same time.

What if you want to watch the same IPTV on your Kodi box or different TV box? Or have the same IPTV on multiple TVs and watch at the same time? Or even share it with friends/relatives? Is it even possible? Eventually, it is!

This application works as a gateway/proxy to the stalker portal. Basically define your stalker authentication details in this app, start it and make all your devices to use this app's address as a stalker portal. This way you can use unlimited amount of devices with a single account.

This app is heavily work-in-progress and many critical things are not done yet:
* Config file. Hardcoding settings is just ugly solution.
* Binary. Who wants to run source code, when you can run native binary
* Keep-alive mechanism
* HTTP requests forwarding optimisation. I think I can make it run even faster.
* Figure out how to use EPG from portal with Kodi (stalker addon settings)
* Check possibility to convert from stalker format to XMLTV (M3U playlist), so you can use simple IPTV addon, or just VLC.
* Find easier way to find details rather than using wireshark (and MITM attack)

# How it works

Once this app is started, it immediatelly with your defined stalker portal using defined credentials and starts listening for requests. Then you need to set all your Stalker TV boxes to this app (from TV box perspective, this app is stalker portal now). And that's it - this app forwards pretty much all requests, while ignoring authentication reuests (that are responsible for disconnecting all other tv boxes that uses the same account).

This is how your typical set-tup box works now:
```
[TV box] <--> [stalker portal]
```

And this is how it works with this app:
```
[TV box] <--> |this app| <--> [stalker portal]
[TV box] <--> |        |
[TV box] <--> |        |
[TV box] <--> |        |
```

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
* Login - *anything*
* Password - *anything*
* token - *anything*
* Serial Number - *anything*
* Device ID - *anything*
* Device ID2 - *anything*
* Signature - *anything*
  
*anything* literally means anything. Like `asdfasdf`. Just don't leave empty fields.
  
Restart Kodi - your IPTV should now work in Kodi.