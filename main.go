package main

import (
	"fmt"
	"log"
	"os"
	"rkndelta/downloader"
	"strconv"
	"time"
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
	d, err := downloader.New(fmt.Sprintf("%s://%s:%s@%s", RknScheme, user, pass, RknURL))
	if err != nil {
		log.Fatalf("cant't create downloader: %v", err)
	}
	go d.DumpDownloader(time.Duration(dumpinterval) * time.Minute)
	d.SocilDownloader(time.Duration(socinterval) * time.Minute)

}
