package main

func main() {
	conf := mustParseConfig()
	w := watcher{dir: conf.BilderDir}
	(&w).start()
}
