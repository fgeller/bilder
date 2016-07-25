package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
)

type config struct {
	BilderDir     string `json:"bilder-dir"`
	URLPathPrefix string `json:"url-path-prefix"`
	AccessLog     string `json:"access-log"`
}

var defaultConfig = config{
	BilderDir: "bilder",
}

func mustParseConfig() config {
	f := flag.String("config", "", "JSON config file for bilder.")
	flag.Parse()

	if *f == "" {
		return defaultConfig
	}

	byts, err := ioutil.ReadFile(*f)
	if err != nil {
		log.Fatalf("Failed to read config file %#v err=%v", *f, err)
	}

	var c config
	if err := json.Unmarshal(byts, &c); err != nil {
		log.Fatalf("Failed to unmarshal contents of %#v as config, err=%v", *f, err)
	}

	if c.BilderDir == "" {
		c.BilderDir = defaultConfig.BilderDir
	}

	return c
}
