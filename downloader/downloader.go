package downloader

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tiaguinho/gosoap"
)

type Downloader struct {
	SOAP *gosoap.Client
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
	}, nil

}

type GetdateRes struct {
	Date int `xml:"lastDumpDate"`
}

type Resp struct {
	Result bool   `xml:"result"`
	Zip    []byte `xml:"registerZipArchive"`
}

func FindXMLInZipAndSave(b []byte) (fn string, err error) {
	zipReader, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", err
	}
	for _, zipFile := range zipReader.File {
		if err != nil {
			log.Println(err)
			continue
		}
		if strings.HasSuffix(zipFile.Name, ".xml") {
			fn = zipFile.Name
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
