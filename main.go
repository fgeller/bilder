package main

func main() {
	conf := mustParseConfig()
	albums := make(chan []album, 1)
	w := newWatcher(conf.ReloadDelaySeconds, conf.BilderDir, conf.URLPathPrefix, albums)
	s := newServer(conf.Addr, conf.BilderDir, conf.AccessLog, albums)

	go w.start()
	s.serve()
}
