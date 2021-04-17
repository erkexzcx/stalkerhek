package hls

import (
	"sync"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var GeneratedPlaylist = make(map[string]*Stream)
var GeneratedPlaylistMux = sync.RWMutex{} // Used to freeze all clients until playlist is updated

func NewGeneratedStream(title string, s stalker.Stream) {
	GeneratedPlaylistMux.Lock()
	defer GeneratedPlaylistMux.Unlock()

	stream := &Stream{
		StalkerStream: s,
		Mux:           &sync.Mutex{},
	}
	GeneratedPlaylist[title] = stream
}

func cleanupGeneratedPlaylist() {
	GeneratedPlaylistMux.Lock()
	defer GeneratedPlaylistMux.Unlock()

	for k, v := range GeneratedPlaylist {
		if v.isValid() {
			continue
		}
		delete(GeneratedPlaylist, k)
	}
}
