package proxy

import (
	"io"
	"sync"
)

// Stream contains information about readCloser (resp.Body) and viewers count.
type Stream struct {
	stream  io.ReadCloser
	viewers int
	mux     sync.Mutex
}

var streams map[string]*Stream

func closeUnusedStreams() {
	for _, v := range streams {
		go func(v *Stream) {
			v.mux.Lock()
			defer v.mux.Unlock()

			if v.viewers > 0 {
				return
			}

			if v.stream != nil {
				v.stream.Close()
				v.stream = nil
			}
		}(v)
	}
}
