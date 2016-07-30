package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/handlers"
	uuid "github.com/satori/go.uuid"
)

var (
	cookieBaseName = "session-a2bb9-"
	nada           = struct{}{}
)

type syncFile struct {
	sync.Mutex
	os.File
	f *os.File
}

func (sf *syncFile) Write(p []byte) (n int, err error) {
	sf.Lock()
	defer sf.Unlock()
	return sf.f.Write(p)
}

type server struct {
	http.Server
	sync.RWMutex
	addr         string
	albumUpdates <-chan []album
	dir          string
	accessLog    string
	logFile      *syncFile
	albums       map[string]authHandler
}

func newServer(ad, d, al string, au <-chan []album) *server {
	return &server{addr: ad, dir: d, accessLog: al, albumUpdates: au}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" {
		http.Error(w, "404 page not found", 404)
		return
	}

	sep := strings.Index(r.URL.Path, "/")
	an := r.URL.Path
	if sep > 0 {
		an = r.URL.Path[:sep]
	}

	s.RLock()
	h, ok := s.albums[an]
	s.RUnlock()

	if !ok {
		http.Error(w, "404 page not found", 404)
		return
	}

	p := strings.TrimPrefix(r.URL.Path, an)
	r.URL.Path = p
	h.ServeHTTP(w, r)
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

type authHandler struct {
	sync.Mutex
	handler     http.Handler
	name        string
	user, pass  string
	sessions    map[string]struct{}
	authEnabled bool
}

func (h authHandler) isAuthed(sid string) bool {
	h.Lock()
	_, ok := h.sessions[sid]
	h.Unlock()
	return ok
}

func (h authHandler) newSession() string {
	sid := uuid.NewV1().String()
	h.Lock()
	h.sessions[sid] = nada
	h.Unlock()
	return sid
}

func (h authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.authEnabled {
		h.handler.ServeHTTP(w, r)
		return
	}

	cookie, err := r.Cookie(cookieBaseName + h.name)
	if err == nil && cookie != nil && h.isAuthed(cookie.Value) {
		h.handler.ServeHTTP(w, r)
		return
	}

	u, p, ok := r.BasicAuth()
	if !(ok && u == h.user && p == h.pass) {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
		http.Error(w, "Not Authorized", http.StatusUnauthorized)
		return
	}

	sid := h.newSession()
	http.SetCookie(w, &http.Cookie{Name: cookieBaseName + h.name, Value: sid, MaxAge: 0})
	h.handler.ServeHTTP(w, r)
}

func (s *server) listenForUpdates() {
	for as := range s.albumUpdates {
		s.RLock()
		oldHandlers := s.albums
		s.RUnlock()
		hs := make(map[string]authHandler)
		for _, a := range as {
			oh, oldExists := oldHandlers[a.name]
			sess := map[string]struct{}{}
			if oldExists {
				sess = oh.sessions
			}
			h := authHandler{
				handler:     http.FileServer(http.Dir(filepath.Join(s.dir, a.name))),
				name:        a.name,
				user:        a.user,
				pass:        a.pass,
				sessions:    sess,
				authEnabled: a.hasAuth(),
			}
			if s.logFile != nil {
				h.handler = handlers.CombinedLoggingHandler(s.logFile, h.handler)
			}
			hs[a.name] = h
		}

		s.Lock()
		s.albums = hs
		s.Unlock()
	}
}

func (s *server) serve() {
	go s.listenForUpdates()

	if s.accessLog != "" {
		lf, err := os.OpenFile(s.accessLog, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Cannot open access log %#v with write access, err=%v", s.accessLog, err)
		} else {
			s.logFile = &syncFile{f: lf}
			defer lf.Close()
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

	mux.Handle("/b/", http.StripPrefix("/b/", s))

	s.Addr = s.addr
	s.Handler = mux

	log.Printf("Serving on http://" + s.addr)
	log.Fatal(s.ListenAndServe())
}
