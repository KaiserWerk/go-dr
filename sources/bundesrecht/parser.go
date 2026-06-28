package bundesrecht

import (
	"encoding/xml"
	"errors"
	"regexp"
	"strings"
	"time"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/internal/normref"
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
		Jurabk       string `xml:"jurabk"`
		Enbez        string `xml:"enbez"`
		Titel        string `xml:"titel"`
		Date         string `xml:"datum"`
		Ausf         string `xml:"ausfertigung-datum"`
		Ausf2        string `xml:"ausfertigungdatum"`
		Inkraft      string `xml:"inkrafttreten"`
		Inkraft2     string `xml:"inkrafttreten-datum"`
		Inkraft3     string `xml:"inkrafttreten_datum"`
		Ausserkraft  string `xml:"ausserkrafttreten"`
		Ausserkraft2 string `xml:"ausserkrafttreten-datum"`
		Ausserkraft3 string `xml:"ausserkrafttreten_datum"`
		Standangabe  string `xml:"standangabe"`
		Glied        struct {
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
	var publishedAt *time.Time
	var effectiveFrom *time.Time
	var effectiveTo *time.Time
	for _, n := range in.Norms {
		if shortTitle == "" {
			shortTitle = strings.TrimSpace(n.Meta.Jurabk)
		}

		if publishedAt == nil {
			publishedAt = firstParsedDate(
				n.Meta.Ausf,
				n.Meta.Ausf2,
				n.Meta.Date,
				n.Meta.Standangabe,
			)
		}
		if effectiveFrom == nil {
			effectiveFrom = firstParsedDate(
				n.Meta.Inkraft,
				n.Meta.Inkraft2,
				n.Meta.Inkraft3,
			)
		}
		if effectiveTo == nil {
			effectiveTo = firstParsedDate(
				n.Meta.Ausserkraft,
				n.Meta.Ausserkraft2,
				n.Meta.Ausserkraft3,
			)
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
			References: normref.Extract(heading+"\n"+content, shortTitle),
			NormChains: normref.ExtractChains(heading+"\n"+content, shortTitle),
		})
	}

	title := shortTitle
	if title == "" && len(sections) > 0 {
		title = sections[0].Heading
	}

	return &godr.LegalDocument{
		Title:         strings.TrimSpace(title),
		ShortTitle:    strings.TrimSpace(shortTitle),
		Jurisdiction:  "DE-BUND",
		Type:          godr.DocumentTypeLaw,
		PublishedAt:   publishedAt,
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   effectiveTo,
		Sections:      sections,
	}, nil
}

var anyDatePattern = regexp.MustCompile(`\b(?:\d{4}-\d{2}-\d{2}|\d{1,2}\.\d{1,2}\.\d{4}|\d{8})\b`)

func firstParsedDate(values ...string) *time.Time {
	for _, v := range values {
		if t := parseDateLoose(v); t != nil {
			return t
		}
	}
	return nil
}

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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
