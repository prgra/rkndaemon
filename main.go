package main

import (
	"log"

	"github.com/prgra/rkndaemon/daemon"
)

func main() {
	var cfg daemon.Config
	err := cfg.Load()
	if err != nil {
		log.Fatalf("cant't load config: %v", err)
	}
	app, err := daemon.New(cfg)
	if err != nil {
		log.Fatalf("cant't create daemon: %v", err)
	}
	app.Run()

}
