package parser

import (
	"encoding/xml"
	"errors"
	"regexp"
	"strings"
	"time"

	godr "github.com/KaiserWerk/go-dr"
)

// XMLDocumentParser parses basic legal XML into a LegalDocument.
type XMLDocumentParser struct{}

type xmlLaw struct {
	XMLName       xml.Name     `xml:"law"`
	ID            string       `xml:"id,attr"`
	Title         string       `xml:"title"`
	ShortTitle    string       `xml:"shortTitle"`
	PublishedAt   string       `xml:"publishedAt"`
	EffectiveFrom string       `xml:"effectiveFrom"`
	EffectiveTo   string       `xml:"effectiveTo"`
	Jurisdiction  string       `xml:"jurisdiction"`
	Type          string       `xml:"type"`
	Sections      []xmlSection `xml:"sections>section"`
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
	doc.PublishedAt = parseDateLoose(x.PublishedAt)
	doc.EffectiveFrom = parseDateLoose(x.EffectiveFrom)
	doc.EffectiveTo = parseDateLoose(x.EffectiveTo)

	for _, s := range x.Sections {
		doc.Sections = append(doc.Sections, godr.Section{
			Identifier: strings.TrimSpace(s.Identifier),
			Heading:    strings.TrimSpace(s.Heading),
			Content:    strings.TrimSpace(s.Content),
		})
	}

	return doc, nil
}

var anyDatePattern = regexp.MustCompile(`\b(?:\d{4}-\d{2}-\d{2}|\d{1,2}\.\d{1,2}\.\d{4}|\d{8})\b`)

func parseDateLoose(v string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}

	for _, layout := range []string{"2006-01-02", "02.01.2006", "2.1.2006", "20060102"} {
		if ts, err := time.Parse(layout, v); err == nil {
			t := ts.UTC()
			return &t
		}
	}

	match := anyDatePattern.FindString(v)
	if match == "" || match == v {
		return nil
	}
	return parseDateLoose(match)
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
