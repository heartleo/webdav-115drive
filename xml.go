package main

import (
	"encoding/xml"
	"io"
)

const (
	// StatusMultiStatus is the HTTP status code for WebDAV multi-status response.
	StatusMultiStatus = 207

	xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

// multistatus represents the DAV:multistatus element.
type multistatus struct {
	XMLName  xml.Name      `xml:"d:multistatus"`
	XmlnsD   string        `xml:"xmlns:d,attr"`
	Response []davResponse `xml:"d:response"`
}

// davResponse represents the DAV:response element.
type davResponse struct {
	Href     string   `xml:"d:href"`
	Propstat propstat `xml:"d:propstat"`
}

// propstat represents the DAV:propstat element.
type propstat struct {
	Prop   prop   `xml:"d:prop"`
	Status string `xml:"d:status"`
}

// prop represents the DAV:prop element with common properties.
type prop struct {
	DisplayName   string        `xml:"d:displayname,omitempty"`
	ContentLength string        `xml:"d:getcontentlength,omitempty"`
	LastMod       string        `xml:"d:getlastmodified,omitempty"`
	GetETag       string        `xml:"d:getetag,omitempty"`
	ResourceType  *resourcetype `xml:"d:resourcetype,omitempty"`
}

// resourcetype represents the DAV:resourcetype element.
type resourcetype struct {
	Collection *struct{} `xml:"d:collection,omitempty"`
}

// xmlEncoder creates an XML encoder for the given writer.
func xmlEncoder(w io.Writer) *xml.Encoder {
	return xml.NewEncoder(w)
}
