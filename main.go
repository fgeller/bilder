package main

func main() {
	conf := mustParseConfig()
	go (&watcher{dir: conf.BilderDir}).start()
	(&server{dir: conf.BilderDir}).start()
}
