package parser

import (
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

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
	HTTPSIPs    List
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
		HTTPSIPs:    make(List),
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
	if item.Domain != "" {
		db.SocDomains.Add(item.Domain)
	}
	for i := range item.Subnets {
		db.SocNets.Add(item.Subnets[i])
	}
}

func (db *DB) ParseEl(item Content) {

	switch item.BlockType {
	case "domain":
		for i := range item.Domain {
			d, _ := idna.ToASCII(item.Domain[i])
			db.Domains.Add(d)
			u, err := url.Parse("http://" + item.Domain[i])
			if err != nil {
				continue
			}
			db.Domains.Add(u.Host)
			d2, err := idna.ToASCII(u.Host)
			if err != nil {
				continue
			}
			db.Domains.Add(d2)
		}
	case "ip":
		for i := range item.IP {
			ip := net.ParseIP(item.IP[i])
			if ip.IsGlobalUnicast() {
				db.BlockedIPs.Add(ip.String())
			}
		}
	case "domain-mask":
		for i := range item.Domain {
			sd := item.Domain[i]
			db.DomainMasks.Add(sd)
			if strings.HasPrefix(sd, "*.") {
				d, _ := idna.ToASCII(strings.TrimPrefix(sd, ".*"))
				db.DomainMasks.Add(d)
			}
		}
	}
	https := false
	for i := range item.URL {
		u, err := url.Parse(item.URL[i])
		if err != nil {
			continue
		}
		d, _ := idna.ToASCII(u.String())
		db.URLs.Add(d)
		mip := net.ParseIP(u.Host)
		if mip.IsGlobalUnicast() {
			db.AllIPs.Add(mip.String())
		}
		if u.Scheme == "https" {
			https = true
		}
		db.URLs.Add(u.String())

		// для кривых урлов
		db.URLs.Add(item.URL[i])
		db.URLs.Add(JSDecodeURI(u.String()))
		rstr := strings.ReplaceAll(u.String(), "%", "%25")
		db.URLs.Add(rstr)

	}

	for i := range item.IP {
		ip := net.ParseIP(item.IP[i])
		if ip.IsGlobalUnicast() {
			db.AllIPs.Add(item.IP[i])
			if https {
				db.HTTPSIPs.Add(ip.String())
			}
			if item.BlockType == "domain-mask" ||
				item.BlockType == "domain" {
				db.BlockedIPs.Add(ip.String())
			}
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
		err2 := os.MkdirAll(dir, 755)
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
	err = db.BlockedIPs.WriteFile(fmt.Sprintf("%s/bloked_ips.txt", dir))
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
	err = db.HTTPSIPs.WriteFile(fmt.Sprintf("%s/https_ips.txt", dir))
	if err != nil {
		return err
	}
	err = db.Domains.MixWriteFile(fmt.Sprintf("%s/all_domains.txt", dir), db.DomainMasks)
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
		err2 := os.MkdirAll(dir, 755)
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
	var arr []string
	for k := range l {
		arr = append(arr, k)
	}
	sort.Strings(arr)
	rn := "\n"
	for i := range arr {
		if i == len(arr)-1 {
			rn = ""
		}
		fmt.Fprintf(f, format+rn, arr[i])
	}
	f.Close()
	return nil
}

func (l List) WriteFile(fn string) error {
	return l.WriteFilef("%s", fn)
}

func (l List) MixWriteFilef(format string, fn string, lists ...List) error {
	newlist := make(List)
	for k := range l {
		newlist.Add(k)
	}
	for i := range lists {
		for k := range lists[i] {
			newlist.Add(k)
		}
	}
	return newlist.WriteFilef(format, fn)
}

func (l List) MixWriteFile(fn string, lists ...List) error {
	return l.MixWriteFilef("%s", fn, lists...)
}

func JSDecodeURI(s string) (r string) {
	r = s
	r = strings.Replace(r, "%20", "+", -1)
	r = strings.Replace(r, "!", "%21", -1)
	r = strings.Replace(r, "'", "%27", -1)
	r = strings.Replace(r, "(", "%28", -1)
	r = strings.Replace(r, ")", "%29", -1)
	r = strings.Replace(r, "*", "%2A", -1)
	return r
}
