package main

import (
	"log"
	"net/http"
)

type server struct {
	http.Server
	dir string
}

func assetsHandler(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[len("/a/"):]
	a, ok := assets[p]
	if !ok {
		log.Printf("Got asset request %v but not available.\n", r.URL.Path)
		http.Error(w, "404 page not found", 404)
		return
	}

	w.Header().Set("Content-Type", a.ContentType)
	w.Write(a.Content)
}

func (s *server) start() {
	mux := http.NewServeMux()
	if len(assets) == 0 {
		log.Printf("Serving assets for assets directory.")
		mux.Handle("/a/", http.StripPrefix("/a/", http.FileServer(http.Dir("assets"))))
	} else {
		log.Printf("Serving assets from memory.")
		mux.HandleFunc("/a/", assetsHandler)
	}
	mux.Handle("/b/", http.StripPrefix("/b/", http.FileServer(http.Dir(s.dir))))
	log.Printf("serving %#v\n", s.dir)

	s.Addr = ":8173"
	s.Handler = mux

	log.Printf("Serving on http://0.0.0.0:8173")
	log.Fatal(s.ListenAndServe())
}
