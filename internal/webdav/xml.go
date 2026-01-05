package webdav

import (
	"encoding/xml"
	"io"
)

const (
	XmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

type MultiStatus struct {
	XMLName  xml.Name      `xml:"d:multistatus"`
	XmlnsD   string        `xml:"xmlns:d,attr"`
	Response []DavResponse `xml:"d:response"`
}

type DavResponse struct {
	Href     string   `xml:"d:href"`
	Propstat PropStat `xml:"d:propstat"`
}

type PropStat struct {
	Prop   Prop   `xml:"d:Prop"`
	Status string `xml:"d:status"`
}

type Prop struct {
	DisplayName   string        `xml:"d:displayname,omitempty"`
	ContentLength string        `xml:"d:getcontentlength,omitempty"`
	LastMod       string        `xml:"d:getlastmodified,omitempty"`
	GetETag       string        `xml:"d:getetag,omitempty"`
	ResourceType  *ResourceType `xml:"d:resourcetype,omitempty"`
}

type ResourceType struct {
	Collection *struct{} `xml:"d:collection,omitempty"`
}

func XmlEncoder(w io.Writer) *xml.Encoder {
	return xml.NewEncoder(w)
}
