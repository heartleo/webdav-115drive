package main

import (
	"encoding/xml"
	"io"
)

const (
	xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

type multiStatus struct {
	XMLName  xml.Name      `xml:"d:multistatus"`
	XmlnsD   string        `xml:"xmlns:d,attr"`
	Response []davResponse `xml:"d:response"`
}

type davResponse struct {
	Href     string   `xml:"d:href"`
	Propstat propStat `xml:"d:propstat"`
}

type propStat struct {
	Prop   prop   `xml:"d:prop"`
	Status string `xml:"d:status"`
}

type prop struct {
	DisplayName   string        `xml:"d:displayname,omitempty"`
	ContentLength string        `xml:"d:getcontentlength,omitempty"`
	LastMod       string        `xml:"d:getlastmodified,omitempty"`
	GetETag       string        `xml:"d:getetag,omitempty"`
	ResourceType  *resourceType `xml:"d:resourcetype,omitempty"`
}

type resourceType struct {
	Collection *struct{} `xml:"d:collection,omitempty"`
}

func xmlEncoder(w io.Writer) *xml.Encoder {
	return xml.NewEncoder(w)
}
