package parser

import (
	"encoding/xml"
	"time"
)

type SocRecord struct {
	XMLName     xml.Name  `xml:"content"`
	ID          int       `xml:"-"`
	IncludeTime time.Time `xml:"-"`
	Hash        string    `xml:"-"`
	Name        string    `xml:"resourceName"`
	Domain      string    `xml:"domain"`
	Subnets     []string  `xml:"ipSubnet"`
}
