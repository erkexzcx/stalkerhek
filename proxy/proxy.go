package proxy

import (
	"net/http"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var portal *stalker.Portal

func Start(p *stalker.Portal, bind string) {
	portal = p

	http.HandleFunc("/", requestHandler)

	panic(http.ListenAndServe(bind, nil))
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(portal.Model))
}
