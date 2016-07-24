package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nfnt/resize"
)

type imgDetails struct {
	Thumb     string
	Width     int
	Height    int
	Caption   string
	Path      string
	ThumbPath string
}

type dirDetails struct {
	Title  string
	Images []*imgDetails
}

type watcher struct {
	dir     string
	configs map[string]dirConfig
	images  map[string]map[string]*imgDetails
}
type dirConfig struct {
	Title    string
	Captions map[string]string
}

var (
	thumbRegexp     = regexp.MustCompile("(?i)^(.+)_thumb\\.(jpg|jpeg)$")
	imageRegexp     = regexp.MustCompile("(?i)^(.+)\\.(jpg|jpeg)$")
	dirConfigRegexp = regexp.MustCompile("(?i)^bilder.json$")
)

func (w *watcher) start() {
	for {
		w.reloadContents()
		w.ensureThumbs()
		w.writeIndexes()
		<-time.After(10 * time.Second)
	}
}

type byImgName []*imgDetails

func (a byImgName) Len() int           { return len(a) }
func (a byImgName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byImgName) Less(i, j int) bool { return a[i].Path < a[j].Path }

func (w *watcher) writeIndexes() {
	tmpl := template.Must(template.New("dirIndex").Parse(dirIndexTempl))
	for d, is := range w.images {
		p := filepath.Join(w.dir, d, "index.html")
		var ids []*imgDetails
		for _, id := range is {
			ids = append(ids, id)
		}
		sort.Sort(byImgName(ids))
		title := d
		if cfg, exists := w.configs[d]; exists && cfg.Title != "" {
			title = cfg.Title
		}
		dd := dirDetails{
			Title:  title,
			Images: ids,
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, dd); err != nil {
			log.Printf("Failed to execute index template for %#v, err=%v\n", d, err)
			return
		}

		if err := ioutil.WriteFile(p, buf.Bytes(), 0644); err != nil {
			log.Printf("Failed to write index.html for %#v, err=%v\n", d, err)
			return
		}
	}
}

func (w *watcher) ensureThumbs() {
	for d, is := range w.images {
		for i, id := range is {
			if id.Thumb == "" {
				tn, err := w.generateThumb(d, i)
				if err != nil {
					log.Printf("Failed to generate thumb for %#v in %#v, err=%v", i, d, err)
					continue
				}
				id.ThumbPath = strings.Join([]string{"b", d, tn}, "/")
				log.Printf("Generated thumb for %#v in %#v.", i, d)
			}
		}
	}
}

func (w *watcher) generateThumb(d, n string) (string, error) {
	p := filepath.Join(w.dir, d, n)
	ih, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer ih.Close()

	matches := imageRegexp.FindAllStringSubmatch(n, -1)
	base, ending := matches[0][1], matches[0][2]
	tn := base + "_thumb." + ending
	tp := filepath.Join(w.dir, d, tn)
	th, err := os.Create(tp)
	if err != nil {
		return "", err
	}
	defer func() {
		th.Close()
		fmt.Printf("Writing thumb to %v\n", tp)
	}()

	img, err := jpeg.Decode(ih)
	if err != nil {
		return "", err
	}

	thumb := resize.Thumbnail(200, 200, img, resize.Lanczos3)

	return tn, jpeg.Encode(th, thumb, nil)
}

type byName []os.FileInfo

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name() < a[j].Name() }

func (w *watcher) reloadContents() {
	log.Printf("Reloading contents of %#v.", w.dir)
	ds, err := ioutil.ReadDir(w.dir)
	if err != nil {
		log.Printf("Failed to read contents of %#v, err=%v", w.dir, err)
		return
	}

	w.images = map[string]map[string]*imgDetails{}
	for _, d := range ds {
		if d.IsDir() {
			p := filepath.Join(w.dir, d.Name())
			fs, err := ioutil.ReadDir(p)
			if err != nil {
				log.Printf("Failed to read contents of %#v, err=%v", p, err)
				continue
			}
			sort.Sort(byName(fs)) // sort so thumbs always appear after img

			// find config first, to load captions
			for _, f := range fs {
				switch {
				case f.IsDir() || f.Size() == 0 || time.Since(f.ModTime()) < (200*time.Millisecond):
					continue
				case dirConfigRegexp.MatchString(f.Name()):
					fp := filepath.Join(p, f.Name())
					byts, err := ioutil.ReadFile(fp)
					if err != nil {
						log.Printf("Failed to read dir config %#v, err=%v", fp, err)
						continue
					}
					var cfg dirConfig
					if err = json.Unmarshal(byts, &cfg); err != nil {
						log.Printf("Failed to unmarshal dir config %#v, err=%v", fp, err)
						continue
					}
					if w.configs == nil {
						w.configs = map[string]dirConfig{d.Name(): cfg}
					} else {
						w.configs[d.Name()] = cfg
					}
				}
			}

			// find images and possibly thumbs
			for _, f := range fs {
				switch {
				case f.IsDir() || f.Size() == 0 || time.Since(f.ModTime()) < (10*time.Second):
					continue
				case thumbRegexp.MatchString(f.Name()):
					_, dirExists := w.images[d.Name()]
					if !dirExists {
						log.Printf("Unexpected thumb image %#v in %#v", f.Name(), d.Name())
						continue
					}
					matches := thumbRegexp.FindAllStringSubmatch(f.Name(), -1)
					base, ending := matches[0][1], matches[0][2]
					img := base + "." + ending
					_, imgExists := w.images[d.Name()][img]
					if !imgExists {
						log.Printf("Unexpected thumb image %#v in %#v", f.Name(), d.Name())
						continue
					}
					w.images[d.Name()][img].Thumb = f.Name()
					w.images[d.Name()][img].ThumbPath = strings.Join([]string{"b", d.Name(), f.Name()}, "/")

				case imageRegexp.MatchString(f.Name()):
					p := filepath.Join(w.dir, d.Name(), f.Name())
					fh, err := os.Open(p)
					if err != nil {
						log.Printf("Failed to read %#v for details, err=%v", p, err)
						continue
					}
					img, _, err := image.DecodeConfig(fh)
					if err != nil {
						log.Printf("Failed to decode %#v for details, err=%v", p, err)
					}

					var cptn string
					cfg, cfgExists := w.configs[d.Name()]
					if cfgExists {
						cptn = cfg.Captions[f.Name()]
					}

					details := &imgDetails{
						Width:   img.Width,
						Height:  img.Height,
						Caption: cptn,
						Path:    strings.Join([]string{"b", d.Name(), f.Name()}, "/"),
					}
					if _, dirExists := w.images[d.Name()]; dirExists {
						w.images[d.Name()][f.Name()] = details
					} else {
						w.images[d.Name()] = map[string]*imgDetails{f.Name(): details}
					}
				}
			}
		}
	}
}

var (
	dirIndexTempl = `<!doctype html>
<html>
    <head>
        <title>{{.Title}}</title>
        <link rel="stylesheet" href="/a/photoswipe.css">
        <link rel="stylesheet" href="/a/default-skin.css">
        <script src="/a/photoswipe.min.js"></script>
        <script src="/a/photoswipe-ui-default.min.js"></script>
        <style>
         body {
             font-family: Roboto, sans-serif;
             margin: 0 10pt;
         }
         h1 {
             color: #212121;
         }
         #gallery-overview figure {
           margin: 10px;
         }
         #gallery-overview figcaption {
             font-size: 9pt;
             font-weight: bold;
             text-align: center;
         }
         #gallery-overview {
             display: flex;
             flex-wrap: wrap;
             justify-content: center;
             align-items: center;
             background-color: #212121;
             color: #fafafa;
             border-radius: 3pt;
         }
        </style>
    </head>
    <body>
        <h1>{{.Title}}</h1>
        <div class="pswp" tabindex="-1" role="dialog" aria-hidden="true">
            <div class="pswp__bg"></div>
            <div class="pswp__scroll-wrap">
                <div class="pswp__container">
                    <div class="pswp__item"></div>
                    <div class="pswp__item"></div>
                    <div class="pswp__item"></div>
                </div>
                <div class="pswp__ui pswp__ui--hidden">
                    <div class="pswp__top-bar">
                        <div class="pswp__counter"></div>
                        <button class="pswp__button pswp__button--close" title="Close (Esc)"></button>
                        <button class="pswp__button pswp__button--share" title="Share"></button>
                        <button class="pswp__button pswp__button--fs" title="Toggle fullscreen"></button>
                        <button class="pswp__button pswp__button--zoom" title="Zoom in/out"></button>
                        <div class="pswp__preloader">
                            <div class="pswp__preloader__icn">
                                <div class="pswp__preloader__cut">
                                    <div class="pswp__preloader__donut"></div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="pswp__share-modal pswp__share-modal--hidden pswp__single-tap">
                        <div class="pswp__share-tooltip"></div>
                    </div>
                    <button class="pswp__button pswp__button--arrow--left" title="Previous (arrow left)">
                    </button>
                    <button class="pswp__button pswp__button--arrow--right" title="Next (arrow right)">
                    </button>
                    <div class="pswp__caption">
                        <div class="pswp__caption__center"></div>
                    </div>
                </div>
            </div>
        </div>
        <div id="gallery-overview" class="gallery-overview">
            {{range .Images}}
            <figure>
                <a href="/{{.Path}}" data-size="{{.Width}}x{{.Height}}">
                    <img src="/{{.ThumbPath}}" width="200" />
                </a>
                <figcaption>{{.Caption}}&nbsp;</figcaption>
            </figure>
            {{end}}
        </div>
        <script>
         var parseThumbnailElements = function(el) {
             var thumbElements = el.childNodes,
                 numNodes = thumbElements.length,
                 items = [],
                 figureEl,
                 linkEl,
                 size,
                 item;
             for(var i = 0; i < numNodes; i++) {
                 figureEl = thumbElements[i];
                 if(figureEl.nodeType !== 1) {
                     continue;
                 }
                 linkEl = figureEl.children[0];
                 size = linkEl.getAttribute('data-size').split('x');
                 item = {
                     src: linkEl.getAttribute('href'),
                     w: parseInt(size[0], 10),
                     h: parseInt(size[1], 10)
                 };
                 if(figureEl.children.length > 1) {
                     item.title = figureEl.children[1].innerHTML;
                 }
                 if(linkEl.children.length > 0) {
                     item.msrc = linkEl.children[0].getAttribute('src');
                 }
                 item.el = figureEl;
                 items.push(item);
             }
             return items;
         };

         // find nearest parent element
         var closest = function closest(el, fn) {
             return el && ( fn(el) ? el : closest(el.parentNode, fn) );
         };

         // triggers when user clicks on thumbnail
         var onThumbnailsClick = function(e) {
             e = e || window.event;
             e.preventDefault ? e.preventDefault() : e.returnValue = false;
             var eTarget = e.target || e.srcElement;
             // find root element of slide
             var clickedListItem = closest(eTarget, function(el) {
                 return (el.tagName && el.tagName.toUpperCase() === 'FIGURE');
             });
             if(!clickedListItem) {
                 return;
             }

             // find index of clicked item by looping through all child nodes
             // alternatively, you may define index via data- attribute
             var clickedGallery = clickedListItem.parentNode,
                 childNodes = clickedListItem.parentNode.childNodes,
                 numChildNodes = childNodes.length,
                 nodeIndex = 0,
                 index;
             for (var i = 0; i < numChildNodes; i++) {
                 if(childNodes[i].nodeType !== 1) {
                     continue;
                 }
                 if(childNodes[i] === clickedListItem) {
                     index = nodeIndex;
                     break;
                 }
                 nodeIndex++;
             }

             if(index >= 0) {
                 openPhotoSwipe( index, clickedGallery );
             }
             console.log("couldn't find a valid index, not opening");
             return false;
         };

         // parse picture index and gallery index from URL (#&pid=1&gid=2)
         var photoswipeParseHash = function() {
             var hash = window.location.hash.substring(1),
                 params = {};

             if(hash.length < 5) {
                 return params;
             }

             var vars = hash.split('&');
             for (var i = 0; i < vars.length; i++) {
                 if(!vars[i]) {
                     continue;
                 }
                 var pair = vars[i].split('=');
                 if(pair.length < 2) {
                     continue;
                 }
                 params[pair[0]] = pair[1];
             }

             if(params.gid) {
                 params.gid = parseInt(params.gid, 10);
             }

             return params;
         };

         var openPhotoSwipe = function(index, galleryElement, disableAnimation, fromURL) {
             var pswpElement = document.querySelectorAll('.pswp')[0],
                 gallery,
                 options,
                 items;
             items = parseThumbnailElements(galleryElement);
             options = {
                 galleryUID: galleryElement.getAttribute('data-pswp-uid'),
                 getThumbBoundsFn: function(index) {
                     var thumbnail = items[index].el.getElementsByTagName('img')[0],
                         pageYScroll = window.pageYOffset || document.documentElement.scrollTop,
                         rect = thumbnail.getBoundingClientRect();
                     return {x:rect.left, y:rect.top + pageYScroll, w:rect.width};
                 },
                 shareButtons: [
                     {id:'download', label:'Download image', url:'{{"{{"}}raw_image_url{{"}}"}}', download:true}
                 ]
             };

             if(fromURL) {
                 if(options.galleryPIDs) {
                     for(var j = 0; j < items.length; j++) {
                         if(items[j].pid == index) {
                             options.index = j;
                             break;
                         }
                     }
                 } else {
                     options.index = parseInt(index, 10) - 1;
                 }
             } else {
                 options.index = parseInt(index, 10);
             }

             // exit if index not found
             if( isNaN(options.index) ) {
                 console.log("couldn't find index in open")
                 return;
             }

             if(disableAnimation) {
                 options.showAnimationDuration = 0;
             }

             gallery = new PhotoSwipe( pswpElement, PhotoSwipeUI_Default, items, options);
             gallery.init();
         };

         var initPhotoSwipeFromDOM = function(gallerySelector) {
             var galleryElements = document.querySelectorAll( gallerySelector );
             for(var i = 0, l = galleryElements.length; i < l; i++) {
                 galleryElements[i].setAttribute('data-pswp-uid', i+1);
                 galleryElements[i].onclick = onThumbnailsClick;
             }
         };
         initPhotoSwipeFromDOM('.gallery-overview');
        </script>
    </body>
</html>
`
)
