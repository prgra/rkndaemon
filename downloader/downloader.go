package downloader

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"rkndelta/parser"
	"strconv"
	"strings"
	"time"

	"github.com/tiaguinho/gosoap"
)

type Downloader struct {
	SOAP *gosoap.Client
	DB   *parser.DB
}

func New(endpoint string) (*Downloader, error) {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	soap, err := gosoap.SoapClient(endpoint, httpClient)
	if err != nil {
		return nil, err
	}

	soap.Username = u.User.Username()
	soap.Password, _ = u.User.Password()
	return &Downloader{
		SOAP: soap,
		DB:   parser.NewDB(),
	}, nil

}

type GetdateRes struct {
	Date int `xml:"lastDumpDate"`
}

type Resp struct {
	Result bool   `xml:"result"`
	Zip    []byte `xml:"registerZipArchive"`
}

func findXMLInZipAndSave(b []byte) (fn string, err error) {
	zipReader, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", err
	}
	for _, zipFile := range zipReader.File {
		if err != nil {
			log.Println(err)
			continue
		}
		fn = zipFile.Name
		if strings.HasSuffix(fn, ".xml") {
			fmt.Println("found xml file", fn, ByteCountIEC(int64(zipFile.UncompressedSize64)))
			f, err := os.Create(fn)
			if err != nil {
				return "", err
			}
			unzippedFileBytes, err := readZipFile(zipFile)
			if err != nil {
				return "", err

			}
			r := bytes.NewReader(unzippedFileBytes)
			_, err = io.Copy(f, r)
			if err != nil {
				return "", err
			}
			f.Close()
		}
	}
	return fn, nil
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func (d *Downloader) SocialDownloader(i time.Duration) {
	for {
		res, err := d.SOAP.Call("getResultSocResources", gosoap.Params{})
		if err != nil {
			log.Fatalf("aaaaaa error: %s", err)
		}
		var r Resp
		res.Unmarshal(&r)
		b, err := base64.StdEncoding.DecodeString(string(r.Zip))
		if err != nil {
			panic(err)
		}
		_, err = findXMLInZipAndSave(b)
		if err != nil {
			panic(err)
		}
		time.Sleep(i)
	}
}

func SaveDumpDate(d int) error {
	f, err := os.Create("/tmp/lastrkndump")
	if err != nil {
		return err
	}
	fmt.Fprint(f, d)
	f.Close()
	return nil
}

func LoadDumpDate() (d int, err error) {
	f, err := os.Open("/tmp/lastrkndump")
	if err != nil {
		return 0, err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return 0, err
	}
	d, err = strconv.Atoi(string(b))
	return d, err
}

func (d *Downloader) DumpDownloader(i time.Duration) {
	dd, _ := LoadDumpDate()
	log.Println("loaded dumpdate", dd, time.Unix(int64(dd/1000), 0))
	for {
		if dd != 0 {
			time.Sleep(i)
		}
		res, err := d.SOAP.Call("getLastDumpDate", nil)
		if err != nil {
			log.Fatalf("Call error: %s", err)
		}
		var rd GetdateRes
		res.Unmarshal(&rd)
		log.Println("got dump date", rd.Date)
		if rd.Date == dd {
			continue
		}
		dd = rd.Date
		err = SaveDumpDate(dd)
		if err != nil {
			panic(err)
		}
		res, err = d.SOAP.Call("getResult", gosoap.Params{})
		if err != nil {
			log.Fatalf("aaaaaa error: %s", err)
		}
		var r Resp
		res.Unmarshal(&r)
		b, err := base64.StdEncoding.DecodeString(string(r.Zip))
		if err != nil {
			panic(err)
		}
		fn, err := findXMLInZipAndSave(b)
		if err != nil {
			panic(err)
		}
		d.DB.ReadDumpFile(fn)
	}
}
