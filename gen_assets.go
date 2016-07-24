package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type asset struct {
	ContentType string
	Content     []byte
}

func main() {
	var (
		assetsFileName = "assets.go"
		assetsTmpl     = `package main

type asset struct {
	ContentType string
	Content     []byte
}

var assets = map[string]asset{}

func init() {
{{ range $n, $a := . }}
	assets["{{ printf "%v" $n }}"] = asset{
		ContentType: "{{ printf "%v"  $a.ContentType }}",
		Content:     {{ printf "%#v"  $a.Content }},
	}
{{ end }}
}
`
	)

	as := loadAssets()
	tmpl := template.Must(template.New("assets").Parse(assetsTmpl))
	fh, err := os.OpenFile(assetsFileName, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	if err := tmpl.Execute(fh, as); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %v assets to %#v", len(as), assetsFileName)
}

func loadAssets() map[string]asset {
	contents, err := ioutil.ReadDir("assets")
	if err != nil {
		log.Fatal(err)
	}

	assets := map[string]asset{}
	for _, fi := range contents {
		if fi.IsDir() {
			continue
		}
		a := asset{ContentType: detectContentType(fi.Name())}
		a.Content, err = ioutil.ReadFile(filepath.Join("assets", fi.Name()))
		if err != nil {
			log.Fatal(err)
		}
		assets[fi.Name()] = a
	}

	return assets
}

func detectContentType(n string) string {
	switch {
	case strings.HasSuffix(n, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(n, ".js"):
		return "application/javascript"
	case strings.HasSuffix(n, ".html"):
		return "text/html; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
