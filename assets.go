package main

type asset struct {
	ContentType string
	Content     []byte
}

var assets = map[string]asset{}

func init() {}
