package hls

import "sync"

// Store temporarily generated links (channels) here
var generatedPlaylist = make(map[string]*Channel)
var generatedPlaylistMux = sync.RWMutex{} // Used to freeze all clients until generatedPlaylist is updated

// func cleanupGeneratedPlaylist() {
// 	// TODO
// }
