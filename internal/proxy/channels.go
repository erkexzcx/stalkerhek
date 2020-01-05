package proxy

import (
	"sync"

	"github.com/erkexzcx/stalkerhek/pkg/stalker"
)

const (
	channelTypeUnknown = 0 // Default value when channel has never been opened
	channelTypeStream  = 1 // Stream that never ends. E.g. application/octet-stream
	channelTypeMedia   = 2 // Media file. E.g. large mp4 file
	channelTypeM3U8    = 3 // M3U8 IPTV
)

// Channel is a wrapper for stalker channel that contains channel type (e.g. stream, M3U8 or media file).
type Channel struct {
	Stalker   *stalker.Channel // Reference to stalker's channel
	chTypeMux sync.RWMutex     // Used to prevent multiple go routines trying to identify channel's type.
	chType    int              // See channelType* consts
}

// Channels list
var channels map[string]*Channel
