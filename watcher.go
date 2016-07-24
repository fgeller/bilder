package main

import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nfnt/resize"
)

type watcher struct {
	dir     string
	configs map[string]dirConfig
	images  map[string]map[string]struct{}
}
type dirConfig struct {
	Title    string
	Captions map[string]string
}

var (
	jpgRegexp       = regexp.MustCompile("(?i)^(.+)\\.(jpg|jpeg)$")
	dirConfigRegexp = regexp.MustCompile("(?i)^bilder.json$")
	nada            = struct{}{}
)

func (w *watcher) start() {
	for {
		select {
		case <-time.After(10 * time.Second):
			w.reloadContents()
			w.ensureThumbs()
		}
	}
}

func (w *watcher) ensureThumbs() {
	for d, is := range w.images {
		for i := range is {
			matches := jpgRegexp.FindAllStringSubmatch(i, -1)
			base, ending := matches[0][1], matches[0][2]
			_, hasThumb := w.images[d][base+"_thumb."+ending]
			isThumb := strings.HasSuffix(matches[0][1], "_thumb")
			if !(isThumb || hasThumb) {
				if err := w.generateThumb(d, i); err != nil {
					log.Printf("Failed to generate thumb for %#v in %#v, err=%v", i, d, err)
					continue
				}
				log.Printf("Generated thumb for %#v in %#v.", i, d)
			}
		}
	}
}

func (w *watcher) generateThumb(d, n string) error {
	p := filepath.Join(w.dir, d, n)
	ih, err := os.Open(p)
	if err != nil {
		return err
	}
	defer ih.Close()

	matches := jpgRegexp.FindAllStringSubmatch(n, -1)
	base, ending := matches[0][1], matches[0][2]
	tp := filepath.Join(w.dir, d, base+"_thumb."+ending)
	th, err := os.Create(tp)
	if err != nil {
		return err
	}
	defer func() {
		th.Close()
		fmt.Printf("Writing thumb to %v\n", tp)
	}()

	img, err := jpeg.Decode(ih)
	if err != nil {
		return err
	}

	thumb := resize.Thumbnail(200, 200, img, resize.Lanczos3)

	return jpeg.Encode(th, thumb, nil)
}

func (w *watcher) reloadContents() {
	log.Printf("Reloading contents of %#v.", w.dir)
	ds, err := ioutil.ReadDir(w.dir)
	if err != nil {
		log.Printf("Failed to read contents of %#v, err=%v", w.dir, err)
		return
	}

	w.images = map[string]map[string]struct{}{}
	for _, d := range ds {
		if d.IsDir() {
			p := filepath.Join(w.dir, d.Name())
			fs, err := ioutil.ReadDir(p)
			if err != nil {
				log.Printf("Failed to read contents of %#v, err=%v", p, err)
				continue
			}
			for _, f := range fs {
				switch {
				case f.IsDir() || f.Size() == 0 || time.Since(f.ModTime()) < (10*time.Second):
					continue
				case jpgRegexp.MatchString(f.Name()):
					_, ok := w.images[d.Name()]
					if ok {
						w.images[d.Name()][f.Name()] = nada // hrm...
					} else {
						w.images[d.Name()] = map[string]struct{}{f.Name(): nada}
					}
				case dirConfigRegexp.MatchString(f.Name()):
					fp := filepath.Join(p, f.Name())
					byts, err := ioutil.ReadFile(fp)
					if err != nil {
						log.Printf("Failed to read dir config %#v, err=%v", fp, err)
						continue
					}
					if err = json.Unmarshal(byts, w.configs[d.Name()]); err != nil {
						log.Printf("Failed to unmarshal dir config %#v, err=%v", fp, err)
						continue
					}
				}
			}
		}
	}
}
