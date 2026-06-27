package bundesrecht

import (
	"encoding/xml"
	"errors"
	"strings"

	"github.com/KaiserWerk/go-dr"
)

// Parser matches the core parser contract while allowing source-level specialization.
type Parser interface {
	Parse(raw []byte) (*godr.LegalDocument, error)
}

// XMLDocumentParser extracts section-level content from Bundesrecht XML.
type XMLDocumentParser struct{}

type bundesrechtDoc struct {
	Norms []bundesrechtNorm `xml:"norm"`
}

type bundesrechtNorm struct {
	DokNR string `xml:"doknr,attr"`
	Meta  struct {
		Jurabk string `xml:"jurabk"`
		Enbez  string `xml:"enbez"`
		Titel  string `xml:"titel"`
		Glied  struct {
			Bez   string `xml:"gliederungsbez"`
			Titel string `xml:"gliederungstitel"`
		} `xml:"gliederungseinheit"`
	} `xml:"metadaten"`
	TextData struct {
		Text struct {
			Content struct {
				Paragraphs []string `xml:"P"`
			} `xml:"Content"`
		} `xml:"text"`
	} `xml:"textdaten"`
}

// Parse implements Parser.
func (p XMLDocumentParser) Parse(raw []byte) (*godr.LegalDocument, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty XML payload")
	}

	var in bundesrechtDoc
	if err := xml.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if len(in.Norms) == 0 {
		return nil, errors.New("no norm entries found")
	}

	sections := make([]godr.Section, 0, len(in.Norms))
	shortTitle := ""
	for _, n := range in.Norms {
		if shortTitle == "" {
			shortTitle = strings.TrimSpace(n.Meta.Jurabk)
		}

		identifier := firstNonEmpty(n.Meta.Enbez, n.Meta.Glied.Bez, n.DokNR)
		heading := firstNonEmpty(n.Meta.Titel, n.Meta.Glied.Titel)
		content := strings.TrimSpace(strings.Join(n.TextData.Text.Content.Paragraphs, "\n"))
		if identifier == "" && heading == "" && content == "" {
			continue
		}

		sections = append(sections, godr.Section{
			Identifier: strings.TrimSpace(identifier),
			Heading:    strings.TrimSpace(heading),
			Content:    content,
		})
	}

	title := shortTitle
	if title == "" && len(sections) > 0 {
		title = sections[0].Heading
	}

	return &godr.LegalDocument{
		Title:        strings.TrimSpace(title),
		ShortTitle:   strings.TrimSpace(shortTitle),
		Jurisdiction: "DE-BUND",
		Type:         godr.DocumentTypeLaw,
		Sections:     sections,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
