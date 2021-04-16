package hls

import "sync"

var GeneratedPlaylist = make(map[string]*Channel)
var GeneratedPlaylistMux = sync.RWMutex{} // Used to freeze all clients until playlist is updated
