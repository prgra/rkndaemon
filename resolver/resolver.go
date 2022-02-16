package resolver

import (
	"log"
	"net"
	"net/url"
	"rkndelta/parser"
	"sync"

	"github.com/bogdanovich/dns_resolver"
)

type Resolver struct {
	inChan      chan url.URL
	outChan     chan []net.IP
	waitGroup   *sync.WaitGroup
	writerWG    *sync.WaitGroup
	dnsResolver *dns_resolver.DnsResolver
}

func New(dnsservers []string) *Resolver {
	dnsresolver := dns_resolver.New(dnsservers)
	dnsresolver.RetryTimes = 3
	return &Resolver{
		inChan:      make(chan url.URL, 1000),
		outChan:     make(chan []net.IP),
		dnsResolver: dnsresolver,
	}
}

func (r Resolver) AddToQueue(url url.URL) {
	r.inChan <- url
}

func (r Resolver) worker() {
	for {
		dom, ok := <-r.inChan
		if !ok {
			r.waitGroup.Done()
			return
		}
		ips, err := r.dnsResolver.LookupHost(dom.Hostname())
		if err != nil {
			log.Println("LookupHost", err)

		}
		ipsmap := make(map[string]bool)
		for i := range ips {
			if ips[i].To4() != nil {
				ipsmap[ips[i].String()] = true
			}
		}

		ips2, err := net.LookupHost(dom.Hostname())
		if err != nil {
			log.Println("net.LookupHost", err)
		}

		for i := range ips2 {
			a := net.ParseIP(ips2[i])
			if a.To4() != nil {
				ipsmap[ips[i].String()] = true
			}
		}
		var res []net.IP
		for k := range ipsmap {
			ip := net.ParseIP(k)
			if ip.To4() != nil {
				res = append(res, ip)
			}
		}

		r.outChan <- res
	}
}

func (r *Resolver) Run(workerCount int, fn string) {
	r.waitGroup.Add(workerCount)
	r.writerWG.Add(1)
	go r.WriteToFile(fn)
	for i := 0; i < workerCount; i++ {
		go r.worker()
	}
}
func (r *Resolver) Close() {
	close(r.inChan)
	r.waitGroup.Wait()
	close(r.outChan)
	r.writerWG.Wait()
}

func (r *Resolver) WriteToFile(fn string) {
	list := make(parser.List)
	for {
		ips, ok := <-r.outChan
		if !ok {
			break
		}
		for i := range ips {
			list.Add(ips[i].String())
		}
	}
	list.WriteFile(fn)
	r.writerWG.Done()
}
