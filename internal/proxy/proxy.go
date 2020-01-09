package proxy

import (
	"log"
	"strings"
	"time"

	"github.com/erkexzcx/stalkerhek/pkg/stalker"
	"github.com/patrickmn/go-cache"
	"github.com/valyala/fasthttp"
)

var stalkerChannels map[string]*stalker.Channel

// Start starts listening for requests. Eventually it starts a proxy server.
func Start(chs map[string]*stalker.Channel) {
	// Initialize channel lists
	channels = make(map[string]*Channel, len(chs))
	m3u8channels = make(map[string]*M3U8Channel, len(chs))
	//streams = make(map[string]*Stream, len(chs))
	for k, v := range chs {
		channels[k] = &Channel{Stalker: v}
	}

	// Initiate cache
	m3u8TSCache = cache.New(20*time.Second, 5*time.Second) // Store cache for 20seconds and clear every 5 seconds

	log.Println("Started!")

	// the corresponding fasthttp code
	m := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		if strings.HasPrefix(path, "/iptv/") {
			channelHandler(ctx)
		} else if strings.HasPrefix(path, "/iptv") {
			playlistHandler(ctx)
		} else {
			ctx.Error("not found", fasthttp.StatusNotFound)
		}
	}
	fasthttp.ListenAndServe(":8987", m)
}
