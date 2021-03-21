package proxy

import (
	"io"
	"net/http"
)

func handleContentMedia(w http.ResponseWriter, r *http.Request, cr *ContentRequest) {
	resp, err := response(cr.Channel.LinkURL)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	handleEstablishedContentMedia(w, r, cr, resp)
}

func handleEstablishedContentMedia(w http.ResponseWriter, r *http.Request, cr *ContentRequest, resp *http.Response) {
	cr.Channel.Mux.Unlock() // So other clients can watch it too
	addHeaders(resp.Header, w.Header(), true)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	cr.Channel.Mux.Lock() // To prevent runtime error because we use 'defer' to unlock mux
}
