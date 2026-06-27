package parser

import (
	"encoding/xml"
	"errors"
	"strings"

	"github.com/KaiserWerk/go-dr"
)

// XMLDocumentParser parses basic legal XML into a LegalDocument.
type XMLDocumentParser struct{}

type xmlLaw struct {
	XMLName      xml.Name     `xml:"law"`
	ID           string       `xml:"id,attr"`
	Title        string       `xml:"title"`
	ShortTitle   string       `xml:"shortTitle"`
	Jurisdiction string       `xml:"jurisdiction"`
	Type         string       `xml:"type"`
	Sections     []xmlSection `xml:"sections>section"`
}

type xmlSection struct {
	Identifier string `xml:"id,attr"`
	Heading    string `xml:"heading"`
	Content    string `xml:"content"`
}

// Parse implements Parser.
func (p XMLDocumentParser) Parse(raw []byte) (*godr.LegalDocument, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty XML payload")
	}

	var x xmlLaw
	if err := xml.Unmarshal(raw, &x); err != nil {
		return nil, err
	}

	doc := &godr.LegalDocument{
		ID:           strings.TrimSpace(x.ID),
		Title:        strings.TrimSpace(x.Title),
		ShortTitle:   strings.TrimSpace(x.ShortTitle),
		Jurisdiction: strings.TrimSpace(x.Jurisdiction),
		Type:         normalizeDocType(x.Type),
		Sections:     make([]godr.Section, 0, len(x.Sections)),
	}

	for _, s := range x.Sections {
		doc.Sections = append(doc.Sections, godr.Section{
			Identifier: strings.TrimSpace(s.Identifier),
			Heading:    strings.TrimSpace(s.Heading),
			Content:    strings.TrimSpace(s.Content),
		})
	}

	return doc, nil
}

func normalizeDocType(v string) godr.DocumentType {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(godr.DocumentTypeLaw):
		return godr.DocumentTypeLaw
	case string(godr.DocumentTypeOrder):
		return godr.DocumentTypeOrder
	case string(godr.DocumentTypeRuling):
		return godr.DocumentTypeRuling
	default:
		return godr.DocumentTypeUnknown
	}
}
