package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
)

type server struct {
	http.Server
	dir       string
	accessLog string
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
	var (
		al  *os.File
		err error
	)
	if s.accessLog != "" {
		al, err = os.OpenFile(s.accessLog, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Cannot open access log %#v with write access, err=%v", s.accessLog, err)
		} else {
			defer al.Close()
		}
	}

	mux := http.NewServeMux()
	if len(assets) == 0 {
		log.Printf("Serving assets for assets directory.")
		mux.Handle("/a/", http.StripPrefix("/a/", http.FileServer(http.Dir("assets"))))
	} else {
		log.Printf("Serving assets from memory.")
		mux.HandleFunc("/a/", assetsHandler)
	}

	if al == nil {
		mux.Handle("/b/", http.StripPrefix("/b/", http.FileServer(http.Dir(s.dir))))
	} else {
		h := http.FileServer(http.Dir(s.dir))
		h = http.StripPrefix("/b/", h)
		h = handlers.CombinedLoggingHandler(al, h)
		log.Printf("Enabling access log to %#v", s.accessLog)
		mux.Handle("/b/", h)
	}
	log.Printf("serving %#v\n", s.dir)

	s.Addr = ":8173"
	s.Handler = mux

	log.Printf("Serving on http://0.0.0.0:8173")
	log.Fatal(s.ListenAndServe())
}
