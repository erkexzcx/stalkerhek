package hls

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var config *stalker.Config

var Playlist = make(map[string]*Stream) // Main HLS playlist, searchable by IPTV channel title
var PlaylistMux = sync.RWMutex{}        // Used to freeze all clients until playlist is updated
var playlistSorted []string             // Sorted titles list of simple playlist
var PlaylistCMD2Title = make(map[string]string)

// Start starts main routine.
func Start(c *stalker.Config) {
	config = c

	// Generate once
	updatePlaylist(3)

	// Regularily refresh ITV channels
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			updatePlaylist(3)
		}
	}()

	// Regularily cleanup 'GeneratedPlaylist'
	go func() {
		for {
			time.Sleep(15 * time.Minute)
			cleanupGeneratedPlaylist()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/iptv", playlistHandler)
	mux.HandleFunc("/iptv/", channelHandler)
	mux.HandleFunc("/generated/", generatedHandler)

	log.Println("HLS service should be started!")
	panic(http.ListenAndServe(c.HLS.Bind, mux))
}

// This function initiates playlist
func updatePlaylist(counter int) {
	PlaylistMux.Lock()
	defer PlaylistMux.Unlock()

	itvs := config.Portal.RetrieveListOfITVs()
	fmt.Println(*itvs[0])

	if len(itvs) == 0 {
		if counter <= 0 {
			log.Fatalln("failed to update ITV list (HLS playlist) - no ITVs returned")
		} else {
			updatePlaylist(counter - 1)
			return
		}
	}

	newPlaylistSorted := make([]string, 0, len(itvs))

	// Check what exists in new returned ITVs list and does not exist in playlist
	for _, itv := range itvs {
		newPlaylistSorted = append(newPlaylistSorted, itv.Title)
		PlaylistCMD2Title[itv.CMDString] = itv.Title

		_, found := Playlist[itv.Title]
		if !found {
			Playlist[itv.Title] = &Stream{
				StalkerStream: itv,
				Mux:           &sync.Mutex{},
				Title:         itv.Title,
			}
			continue
		}
	}

	// Now check what does not exist in returned ITVs list that exists in playlist
	for k, v := range Playlist {

		found := false
		for _, itv := range itvs {
			if k == itv.Title {
				found = true
				break
			}
		}

		if found {
			continue
		}

		delete(Playlist, k)
		delete(PlaylistCMD2Title, v.StalkerStream.CMD())
	}

	sort.Strings(newPlaylistSorted)
	playlistSorted = newPlaylistSorted
}
