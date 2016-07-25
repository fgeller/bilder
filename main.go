package main

func main() {
	conf := mustParseConfig()
	go (&watcher{dir: conf.BilderDir, urlPathPrefix: conf.URLPathPrefix}).start()
	(&server{dir: conf.BilderDir}).start()
}
