// Package daemon for downloading and parsing RKN data
package daemon

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigtoml"
	"github.com/prgra/rkndaemon/downloader"
	"github.com/prgra/rkndaemon/parser"
	"github.com/prgra/rkndaemon/resolver"

	"github.com/tiaguinho/gosoap"
	"golang.org/x/text/encoding/charmap"
)

// App main application
type App struct {
	Downloader *downloader.Downloader
	Resolver   *resolver.Resolver
	Parser     *parser.DB
	Config     Config
	waitGroup  *sync.WaitGroup
}

// Config for application
type Config struct {
	URL            string   `default:"https://vigruzki2.rkn.gov.ru/services/OperatorRequest2/?wsdl" toml:"rknurl" env:"URL"`
	User           string   `toml:"rknuser" env:"USER"`
	Pass           string   `toml:"rknpass" env:"PASS"`
	DNSServers     []string `default:"[8.8.8.8],[1.1.1.1]" toml:"dnses" env:"DNSSERVERS"`
	WorkerCount    int      `default:"64" toml:"dnsworkers" env:"WORKERCOUNT"`
	ResolverFile   string   `default:"output/resolved.txt" toml:"resolvfile" env:"RESOLVERFILE"`
	SocialInterval int      `default:"60" toml:"socinterval" env:"SOCIALINTERVAL"`
	DumpInterval   int      `default:"5" toml:"dumpinterval" env:"DUMPINTERVAL"`
	PostScript     string   `toml:"postscript" env:"POSTSCRIPT"`
	SocialScript   string   `toml:"socialscript" env:"SOCIALSCRIPT"`
	UseDump        bool     `default:"true" toml:"usedump" env:"USEDUMP"`
	UseSoc         bool     `default:"true" toml:"usesoc" env:"USESOC"`
	UseResolver    bool     `default:"false" toml:"useresolver" env:"USERESOLVER"`
	Cron           bool     `dafault:"false" toml:"cron" ENV:"CRON"`
	ListerHTTP     string   `default:"" toml:"listen" ENV:"LISTEN"`
	HTTPToken      string   `default:"" toml:"httptoken" ENV:"HTTPTOKEN"`
}

// Load configuration
func (c *Config) Load() error {
	loader := aconfig.LoaderFor(c, aconfig.Config{
		SkipFlags: true,
		EnvPrefix: "RKN",
		Files: []string{
			"rkndaemon.toml",
			"/etc/rkndaemon.toml",
		},
		FileDecoders: map[string]aconfig.FileDecoder{
			".toml": aconfigtoml.New(),
		},
	})
	err := loader.Load()
	if err != nil {
		return err
	}
	if c.User == "" || c.Pass == "" {
		return fmt.Errorf("need user and password params")
	}
	preu, err := url.Parse(c.URL)
	if err != nil {
		return err
	}
	u, _ := url.Parse(fmt.Sprintf("%s://%s:%s@%s%s?%s", preu.Scheme, c.User, c.Pass, preu.Host, preu.Path, preu.RawQuery))
	c.URL = u.String()
	return nil
}

// New create new application
func New(c Config) (a *App, err error) {
	dwn, err := downloader.New(c.URL)
	if err != nil {
		return a, err
	}
	var wg sync.WaitGroup
	res := resolver.New(c.DNSServers)
	res.Run(c.WorkerCount, c.ResolverFile)
	return &App{
		Downloader: dwn,
		Resolver:   res,
		Parser:     parser.NewDB(),
		Config:     c,
		waitGroup:  &wg,
	}, nil
}

// Run application
func (a *App) Run() {
	if a.Config.UseDump {
		a.waitGroup.Add(1)
		go a.DumpDownloader(time.Duration(a.Config.DumpInterval) * time.Minute)
	}
	if a.Config.UseSoc {
		a.waitGroup.Add(1)
		go a.SocialDownloader(time.Duration(a.Config.SocialInterval) * time.Minute)
	}

	if a.Config.ListerHTTP != "" && !a.Config.Cron {
		go func() {
			log.Println("start http server on")
			err := http.ListenAndServe(a.Config.ListerHTTP, a.AuthMiddleware(http.FileServer(http.Dir("output")))) // nolint
			if err != nil {
				log.Fatalf("can't listen http %v", err)
			}
		}()
	}
	a.waitGroup.Wait()
}

// ReadDumpFile read dump file and parse it
func (a *App) ReadDumpFile(fn string) error {
	log.Println("start read dumpfile")
	xmlFile, err := os.Open(path.Clean(fn))
	if err != nil {
		return err
	}
	defer xmlFile.Close()
	xmlDec := xml.NewDecoder(xmlFile)
	xmlDec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "windows-1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		default:
			return nil, fmt.Errorf("unknown charset: %s", charset)
		}
	}
	for {
		t, err := xmlDec.Token()

		if t == nil {
			break
		}
		if err != nil {
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			var item parser.Content
			err = xmlDec.DecodeElement(&item, &se)
			if err != nil &&
				err.Error() != "expected element type <content> but have <register>" {
				fmt.Fprintln(os.Stderr, err)
			}
			for i := range se.Attr {
				if se.Attr[i].Name.Local == "entryType" {
					item.EntityType = se.Attr[i].Value
				}
				if se.Attr[i].Name.Local == "blockType" {
					item.BlockType = se.Attr[i].Value
				}
			}
			a.Parser.ParseEl(item)
		}
	}
	log.Println("end read dumpfile")

	return nil
}

// ReadSocialFile read social file and parse it
func (a *App) ReadSocialFile(fn string) error {
	log.Println("start read social")
	xmlFile, err := os.Open(path.Clean(fn))
	if err != nil {
		return err
	}
	defer xmlFile.Close()
	xmlDec := xml.NewDecoder(xmlFile)
	for {
		t, xerr := xmlDec.Token()

		if t == nil {
			break
		}
		if xerr != nil {
			return xerr
		}
		switch se := t.(type) {
		case xml.StartElement:
			var item parser.SocRecord
			err = xmlDec.DecodeElement(&item, &se)
			if err != nil &&
				err.Error() != "expected element type <content> but have <registerSocResources>" {
				fmt.Fprintln(os.Stderr, err)
			}
			for i := range se.Attr {
				// id="2" hash="ffdb3ec46de4883efd3c1ca99d7c0ee0" includeTime="2022-01-26T22:00:00+03:00"

				if se.Attr[i].Name.Local == "id" {
					item.ID, _ = strconv.Atoi(se.Attr[i].Value)
				}
				if se.Attr[i].Name.Local == "hash" {
					item.Hash = se.Attr[i].Value
				}
				if se.Attr[i].Name.Local == "includeTime" {
					item.IncludeTime, _ = time.Parse("2006-01-02T15:04:05-07:00", se.Attr[i].Value)

				}
			}
			a.Parser.ParseSoc(item)
		}
	}
	err = a.Parser.WriteSocialFiles("output")
	if err != nil {
		return err
	}
	log.Println("end read social file")
	return nil
}

// DumpDownloader download dump
func (a *App) DumpDownloader(i time.Duration) {
	dd, _ := downloader.LoadDumpDate()
	log.Println("loaded dumpdate", dd, time.Unix(int64(dd/1000), 0))
	for {
		if dd != 0 && !a.Config.Cron {
			time.Sleep(i)
		}
		res, err := a.Downloader.SOAP.Call("getLastDumpDate", nil)
		if err != nil {
			log.Printf("Call getLastDumpDate: %s", err)
			time.Sleep(30 * time.Second)
			continue
		}
		var rd downloader.GetdateRes
		err = res.Unmarshal(&rd)
		if err != nil {
			log.Println("unmarshal", err)
			time.Sleep(30 * time.Second)
			continue
		}
		log.Println("got dump date", rd.Date, time.Unix(int64(dd/1000), 0))
		if rd.Date == dd && !a.Config.Cron {
			continue
		}
		dd = rd.Date
		err = downloader.SaveDumpDate(dd)
		if err != nil {
			log.Println("can't save dumpdate", err)
			time.Sleep(30 * time.Second)
			continue
		}
		res, err = a.Downloader.SOAP.Call("getResult", gosoap.Params{})
		if err != nil {
			log.Printf("Call getResult: %s", err)
			time.Sleep(30 * time.Second)
			continue
		}
		var r downloader.Resp
		err = res.Unmarshal(&r)
		if err != nil {
			log.Println("unmarshal", err)
			time.Sleep(30 * time.Second)
			continue
		}
		b, err := base64.StdEncoding.DecodeString(string(r.Zip))
		if err != nil {
			log.Println("can't unmarshal", err)
			time.Sleep(30 * time.Second)
			continue
		}
		fn, err := downloader.FindXMLInZipAndSave(b)
		if err != nil {
			log.Println("FindXMLInZipAndSave", err)
			time.Sleep(30 * time.Second)
			continue
		}
		err = a.ReadDumpFile(fn)
		if err != nil {
			log.Println("ReadDumpFile", err)
			time.Sleep(30 * time.Second)
			continue
		}
		err = a.Parser.WriteFiles("output")
		if err != nil {
			log.Println("WriteFiles", err)
			time.Sleep(30 * time.Second)
			continue
		}

		if a.Config.UseResolver {
			a.Resolve()
		}
		if a.Config.PostScript != "" &&
			!strings.ContainsAny(a.Config.PostScript, "|;`*?") {
			cmd := exec.Command(path.Clean(a.Config.PostScript)) // nolint
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("PostScript", err)
			}
			log.Println("PostScript", string(out))
		}
		if a.Config.Cron {
			fmt.Println("cron detected exit")
			break
		}
	}
	a.waitGroup.Done()
}

// SocialDownloader download social resources
func (a *App) SocialDownloader(i time.Duration) {
	for {
		res, err := a.Downloader.SOAP.Call("getResultSocResources", gosoap.Params{})
		if err != nil {
			if strings.HasPrefix(err.Error(), "XML syntax error") {
				log.Println("are u add server IP to https://service.rkn.gov.ru/monitoring/vigruzka")
			}
			log.Printf("social download error: %s", err)
			time.Sleep(30 * time.Second)
			continue
		}
		var r downloader.Resp
		err = res.Unmarshal(&r)
		if err != nil {
			log.Printf("socialUnmarshal: %s", err)
		}
		b, err := base64.StdEncoding.DecodeString(string(r.Zip))
		if err != nil {
			log.Printf("socialDecodeString: %s", err)
		}
		fn, err := downloader.FindXMLInZipAndSave(b)
		if err != nil {
			log.Printf("socialFindXMLInZipAndSave: %s", err)
		}
		err = a.ReadSocialFile(fn)
		if err != nil {
			log.Printf("socialReadSocialFilee: %s", err)
		}

		if a.Config.SocialScript != "" &&
			!strings.ContainsAny(a.Config.SocialScript, "|;`*?") {
			cmd := exec.Command(path.Clean(a.Config.SocialScript)) // nolint
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("SocialScript", err)
			}
			log.Println("SocialScript", string(out))
		}
		if !a.Config.Cron {
			time.Sleep(i)
		} else {
			fmt.Println("social cron detected exit")
			break
		}

	}
	a.waitGroup.Done()
}

// Resolve all domains from parser
func (a *App) Resolve() {
	log.Printf("start resolving on %d workers", a.Config.WorkerCount)
	t := time.Now()
	cnt := 0
	pps := 0
	all := len(a.Parser.URLs)
	skip := 0
	resolved := make(map[string]bool)
	for k := range a.Parser.URLs {
		u, err := url.Parse(k)
		if err != nil {
			continue
		}
		if !resolved[u.Hostname()] {
			a.Resolver.AddToQueue(u)
			skip++
		}
		resolved[u.Hostname()] = true
		cnt++
		pps++
		if time.Since(t) > time.Second*10 {
			log.Printf("resolve speed %d per second, %3.2f%% done, skip %d", pps/10, float64(cnt)/float64(all)*100, skip)
			t = time.Now()
			pps = 0
			skip = 0
		}

	}
	a.Resolver.Close()
	log.Println("end resolving")
	a.Resolver = resolver.New(a.Config.DNSServers)
	a.Resolver.Run(a.Config.WorkerCount, a.Config.ResolverFile)
}
