package main

import (
	"log"
	"net/http"
)

type server struct {
	http.Server
	dir string
}

func (s *server) start() {
	mux := http.NewServeMux()
	mux.Handle("/a/", http.StripPrefix("/a/", http.FileServer(http.Dir("assets"))))
	mux.Handle("/b/", http.StripPrefix("/b/", http.FileServer(http.Dir(s.dir))))
	log.Printf("serving %#v\n", s.dir)

	s.Addr = ":8173"
	s.Handler = mux

	log.Printf("Serving on http://0.0.0.0:8173")
	log.Fatal(s.ListenAndServe())
}
