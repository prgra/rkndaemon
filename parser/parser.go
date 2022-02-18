package parser

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os"

	"golang.org/x/net/idna"
)

type Content struct {
	XMLName    xml.Name `xml:"content"`
	Domain     []string `xml:"domain"`
	IP         []string `xml:"ip"`
	IPSubnet   []string `xml:"ipSubnet"`
	URL        []string `xml:"url"`
	BlockType  string   `xml:"-"`
	EntityType string   `xml:"-"`
}

type List map[string]bool

type DB struct {
	WhiteIp     List
	WhiteDomain List
	AllIPs      List
	BlockedIPs  List
	URLs        List
	DomainMasks List
	Domains     List
	Subnets     List
	SocNets     List
	SocDomains  List
}

func NewDB() *DB {
	return &DB{
		WhiteIp:     make(List),
		WhiteDomain: make(List),
		AllIPs:      make(List),
		BlockedIPs:  make(List),
		URLs:        make(List),
		DomainMasks: make(List),
		Domains:     make(List),
		Subnets:     make(List),
		SocNets:     make(List),
		SocDomains:  make(List),
	}
}

func (l List) Add(s string) {
	l[s] = true
}
func (db *DB) ParseSoc(item SocRecord) {
	db.SocDomains.Add(item.Domain)
	for i := range item.Subnets {
		db.SocNets.Add(item.Subnets[i])
	}
}

func (db *DB) ParseEl(item Content) {

	switch item.BlockType {
	case "domain":
		for i := range item.Domain {
			db.URLs.Add(item.Domain[i])
			d, _ := idna.ToASCII(item.Domain[i])
			db.URLs.Add(d)
			u, err := url.Parse("http://" + item.Domain[i])
			if err != nil {
				continue
			}
			db.URLs.Add(u.String())
			db.Domains.Add(u.Host)
			d2, err := idna.ToASCII(u.Host)
			if err != nil {
				continue
			}
			db.Domains.Add(d2)
		}
	case "ip":
		for i := range item.IP {
			db.URLs.Add(item.IP[i])
			ip := net.ParseIP(item.IP[i])
			if ip.IsGlobalUnicast() {
				db.BlockedIPs.Add(ip.String())
			}
		}
	}
	https := false
	for i := range item.URL {
		db.URLs.Add(item.URL[i])
		d, _ := idna.ToASCII(item.URL[i])
		db.URLs.Add(d)
		u, err := url.Parse(item.URL[i])
		if err != nil {
			continue
		}
		mip := net.ParseIP(u.Host)
		if mip.IsGlobalUnicast() {
			db.AllIPs.Add(mip.String())
		}
		if u.Scheme == "https" {
			https = true
		}
		db.URLs.Add(u.String())
	}

	for i := range item.IP {
		ip := net.ParseIP(item.IP[i])
		if ip.IsGlobalUnicast() {
			db.BlockedIPs.Add(ip.String())
			db.AllIPs.Add(item.IP[i])
			if https {
				db.BlockedIPs.Add(ip.String())
			}
		}
	}
	if item.BlockType != "domain-mask" {
		for i := range item.Domain {
			db.URLs.Add(item.Domain[i])
			d, _ := idna.ToASCII(item.Domain[i])
			db.URLs.Add(d)
			u, err := url.Parse(item.Domain[i])
			if err != nil {
				continue
			}
			db.DomainMasks.Add(u.String())
		}
	}
	for i := range item.IPSubnet {
		db.Subnets.Add(item.IPSubnet[i])
	}
}

func (db *DB) WriteFiles(dir string) error {
	log.Println("start write files")

	f, err := os.Stat(dir)

	if err != nil {
		err2 := os.MkdirAll(dir, fs.ModeDir)
		if err2 != nil {
			return err2
		}
	}

	if err == nil && !f.IsDir() {
		return fmt.Errorf("file no dir")
	}

	err = db.AllIPs.WriteFile(fmt.Sprintf("%s/allips.txt", dir))
	if err != nil {
		return err
	}
	err = db.BlockedIPs.WriteFile(fmt.Sprintf("%s/ips.txt", dir))
	if err != nil {
		return err
	}
	err = db.URLs.WriteFile(fmt.Sprintf("%s/urls.txt", dir))
	if err != nil {
		return err
	}
	err = db.URLs.WriteFile(fmt.Sprintf("%s/urls.txt", dir))
	if err != nil {
		return err
	}
	err = db.Subnets.WriteFile(fmt.Sprintf("%s/subnets.txt", dir))
	if err != nil {
		return err
	}
	err = db.DomainMasks.WriteFile(fmt.Sprintf("%s/mdoms.txt", dir))
	if err != nil {
		return err
	}
	err = db.Domains.WriteFile(fmt.Sprintf("%s/domains.txt", dir))
	if err != nil {
		return err
	}

	log.Println("end write files")
	return nil
}

func (db *DB) WriteSocialFiles(dir string) error {
	log.Println("start write social files")

	f, err := os.Stat(dir)

	if err != nil {
		err2 := os.MkdirAll(dir, fs.ModeDir)
		if err2 != nil {
			return err2
		}
	}

	if err == nil && !f.IsDir() {
		return fmt.Errorf("file no dir")
	}
	err = db.SocNets.WriteFile(fmt.Sprintf("%s/SocNets.txt", dir))
	if err != nil {
		return err
	}
	err = db.SocDomains.WriteFile(fmt.Sprintf("%s/SocDomains.txt", dir))
	if err != nil {
		return err
	}
	return nil
}

func (l List) WriteFilef(format string, fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		return nil
	}
	for k := range l {
		fmt.Fprintf(f, format, k)
	}
	f.Close()
	return nil
}

func (l List) WriteFile(fn string) error {
	return l.WriteFilef("%s\n", fn)
}
