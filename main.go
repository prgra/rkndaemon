package main

import (
	"fmt"
	"log"
	"os"
	"rkndelta/daemon"
	"strconv"
)

const (
	RknURL    = "vigruzki2.rkn.gov.ru/services/OperatorRequest2/?wsdl"
	RknScheme = "http"
)

func main() {

	user := os.Getenv("RKNUSER")
	pass := os.Getenv("RKNPASS")
	dumpinterval, _ := strconv.Atoi(os.Getenv("RKNDUMP"))
	if dumpinterval == 0 {
		dumpinterval = 5
	}
	socinterval, _ := strconv.Atoi(os.Getenv("RKNSOC"))
	if socinterval == 0 {
		socinterval = 60
	}
	app, err := daemon.New(daemon.Config{
		KknURL:         fmt.Sprintf("%s://%s:%s@%s", RknScheme, user, pass, RknURL),
		WorkerCount:    8,
		DNSServers:     []string{"8.8.8.8", "1.1.1.1"},
		ResolverFile:   "output/resolved.txt",
		SocialInterval: socinterval,
		DumpInterval:   dumpinterval,
	})
	if err != nil {
		log.Fatalf("cant't create daemon: %v", err)
	}
	app.Run()

}
