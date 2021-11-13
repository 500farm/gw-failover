package main

import log "github.com/sirupsen/logrus"

func main() {
	if err := Config.Load(); err != nil {
		log.Fatal(err)
	}

	coll = newCollector()

	go func() {
		if err := StartMetricsServer(); err != nil {
			log.Error(err)
		}
	}()

	if err := WatchLoop(); err != nil {
		log.Fatal(err)
	}
}
