package main

import (
	"encoding/xml"
	"io"
)

const (
	xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

type multistatus struct {
	XMLName  xml.Name      `xml:"d:multistatus"`
	XmlnsD   string        `xml:"xmlns:d,attr"`
	Response []davResponse `xml:"d:response"`
}

type davResponse struct {
	Href     string   `xml:"d:href"`
	Propstat propstat `xml:"d:propstat"`
}

type propstat struct {
	Prop   prop   `xml:"d:prop"`
	Status string `xml:"d:status"`
}

type prop struct {
	DisplayName   string        `xml:"d:displayname,omitempty"`
	ContentLength string        `xml:"d:getcontentlength,omitempty"`
	LastMod       string        `xml:"d:getlastmodified,omitempty"`
	GetETag       string        `xml:"d:getetag,omitempty"`
	ResourceType  *resourcetype `xml:"d:resourcetype,omitempty"`
}

type resourcetype struct {
	Collection *struct{} `xml:"d:collection,omitempty"`
}

func xmlEncoder(w io.Writer) *xml.Encoder {
	return xml.NewEncoder(w)
}
