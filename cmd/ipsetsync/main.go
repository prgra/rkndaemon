package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		usage()
		os.Exit(0)
	}
	clear := true
	if strings.ToLower(os.Getenv("NOCLEAR")) == "true" ||
		os.Getenv("NOCLEAR") == "1" {
		clear = false
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln("can't open file", err)
	}
	fileIPs := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ip := net.ParseIP(scanner.Text())
		if ip.To4() != nil && ip.IsGlobalUnicast() {
			fileIPs[ip.String()] = true
		}
	}
	f.Close()
	fmt.Println("loaded from file", len(fileIPs))
	cmd := exec.Command("ipset", "-L", os.Args[2])
	out, err := cmd.Output()
	if err != nil {
		log.Fatalln("cant read ipset", err)
	}
	strs := strings.Split(string(out), "\n")
	ipsetIPs := make(map[string]bool)
	for i := range strs {
		ip := net.ParseIP(strs[i])
		if ip.To4() != nil {
			ipsetIPs[ip.String()] = true
		}
	}
	fmt.Printf("loaded from ipset %s %d records\n", os.Args[2], len(ipsetIPs))

	cmd = exec.Command("ipset", "restore")
	restPipe, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalln("cant create pipe for ipset", err)
	}
	go func() {
		for k := range fileIPs {
			_, ok := ipsetIPs[k]
			if !ok {
				fmt.Fprintf(restPipe, "add %s %s\n", os.Args[2], k)
			}
		}
		if clear {
			for k := range ipsetIPs {
				_, ok := fileIPs[k]
				if !ok {
					fmt.Fprintf(restPipe, "del %s %s\n", os.Args[2], k)
				}
			}
		}
		restPipe.Close()
	}()
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalln("read output problem", err)
		fmt.Println("got output", string(out))
	}

	cmd = exec.Command("ipset", "-L", os.Args[2])
	out, err = cmd.Output()
	if err != nil {
		log.Fatalln("cant read ipset", err)
	}
	strs = strings.Split(string(out), "\n")

	ipsetIPs = make(map[string]bool)
	for i := range strs {
		ip := net.ParseIP(strs[i])
		if ip.To4() != nil {
			ipsetIPs[ip.String()] = true
		}
	}
	fmt.Printf("after from ipset %s %d records\n", os.Args[2], len(ipsetIPs))

}

func usage() {
	fmt.Println("usage: ipsetsync file ipsetname")
}
