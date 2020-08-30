package proxy

import "net/http"

func logoHandler(w http.ResponseWriter, r *http.Request) {
	cr, err := getContentRequest(w, r, "/logo/")
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	cr.Channel.Mux.Lock()
	defer cr.Channel.Mux.Unlock()

	if len(cr.Channel.LogoCache) == 0 {
		img, contentType, err := download(cr.Channel.Logo)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		cr.Channel.LogoCache = img
		cr.Channel.LogoCacheContentType = contentType
	}

	w.Header().Set("Content-Type", cr.Channel.LogoCacheContentType)
	w.Write(cr.Channel.LogoCache)
}
