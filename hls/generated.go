package hls

import (
	"sync"
)

var GeneratedPlaylist = make(map[string]*Stream)
var GeneratedPlaylistMux = sync.RWMutex{} // Used to freeze all clients until playlist is updated

func NewGeneratedStream(title string, s Stream) {
	GeneratedPlaylistMux.Lock()
	defer GeneratedPlaylistMux.Unlock()

}

// func Generate(cmdHash, requestType string) {
// 	GeneratedPlaylistMux.Lock()
// 	defer GeneratedPlaylistMux.Unlock()

// 	// Generate random hash
// 	p, _ := rand.Prime(rand.Reader, 64)
// 	randomHash := p.String()

// 	// Create new channel
// 	GeneratedPlaylist[randomHash] = &Stream{
// 		StalkerStream: v,
// 		Mux:           &sync.Mutex{},
// 		LinkType:      requestType,
// 	}
// }

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
