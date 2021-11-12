package main

import log "github.com/sirupsen/logrus"

func main() {
	if err := WatchLoop(); err != nil {
		log.Fatal(err)
	}
}
