package proxy

import (
	"bufio"
	"io"
	"log"
	"sync"
	"time"
)

/*
Spaghetti code explanation - you have a single stream, but you have multiple consumers.
What are you choices? Below code has slice of pipe writers and a for loop, which reads
from stream to buffer, then from buffer writes to **each** pipe writer from the slice.

Usage:
1. Create Stream object, and set stream
2. You MUST execute s.Start() method in order to start for loop
3. logic in for loop will delete unused writers and close stream if no longer providing data
4. Provide writers via s.AddWriter(w *io.PipeWriter) method.
*/

// Stream contains information about readCloser (resp.Body) and viewers count.
type Stream struct {
	stream      io.ReadCloser    // stores stream
	subscribers []*io.PipeWriter // Stores writers
	mux         sync.Mutex
}

var streams map[string]*Stream // Store Streams here

// Start starts a stream's mechanism
func (s *Stream) Start() {
	go func(s *Stream) {
		buf := make([]byte, 512)
		var reader *bufio.Reader
		var k int
		var err error
		var wg sync.WaitGroup
		var deleteThese []int // indexes in array
		for {
			s.mux.Lock()

			if len(s.subscribers) == 0 && s.stream == nil {
				log.Println("Waiting...")
				time.Sleep(5 * time.Second) // Wait 5 sec...
				goto fend                   // And continue pooling
			}

			// Nothing to do, but stream open - close it
			if len(s.subscribers) == 0 && s.stream != nil {
				log.Println("Killing stream...")
				s.stream.Close()
				reader = nil
				s.stream = nil
				goto fend // Nothing to do
			}

			// All good, but reader is nil :/
			if len(s.subscribers) > 0 && s.stream != nil && reader == nil {
				log.Println("Creating reader...")
				reader = bufio.NewReader(s.stream)
			}

			// Read data from stream to buffer
			k, err = reader.Read(buf)

			// If error - it means reader has error (or just empty) and no longer functional. Close it and nil
			if err != nil || k == 0 {
				s.stream.Close()
				reader = nil
				s.stream = nil
				goto fend // Nothing to do
			}

			// Write buffer to subscribers
			wg.Add(len(s.subscribers))
			for i := 0; i < len(s.subscribers); i++ {
				go func(buf []byte, s *Stream, i int) {
					n, err := s.subscribers[i].Write(buf)
					if err != nil || n == 0 {
						// Writer is no longer accepting data, so get rid of it
						deleteThese = append(deleteThese, i)
					}
					wg.Done()
				}(buf, s, i)
			}
			wg.Wait()

			// Get rid of writers that are no longer accepting data
			for _, i := range deleteThese {
				removeFromArray(s.subscribers, i)
			}

		fend:
			s.mux.Unlock()
		}
	}(s)
}

// AddWriter adds a http response writer to the list
func (s *Stream) AddWriter(w *io.PipeWriter) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.subscribers = append(s.subscribers, w)
}

func removeFromArray(s []*io.PipeWriter, i int) []*io.PipeWriter {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}
