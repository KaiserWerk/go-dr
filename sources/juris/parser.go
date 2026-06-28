package juris

import (
	"regexp"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/internal/normref"
	coreparser "github.com/KaiserWerk/go-dr/parser"
)

var shortTitlePattern = regexp.MustCompile(`\(([A-Za-z0-9.\-\s]{2,24})\)`)

// Parser matches the core parser contract while allowing source-level specialization.
type Parser interface {
	Parse(raw []byte) (*godr.LegalDocument, error)
}

// HTMLDocumentParser parses Juris portal pages and enriches sections with normalized references.
type HTMLDocumentParser struct {
	Base         coreparser.HTMLDocumentParser
	Jurisdiction string
}

// Parse implements Parser.
func (p HTMLDocumentParser) Parse(raw []byte) (*godr.LegalDocument, error) {
	doc, err := p.Base.Parse(raw)
	if err != nil {
		return nil, err
	}

	shortTitle := detectShortTitle(doc.Title)
	for i := range doc.Sections {
		text := strings.TrimSpace(doc.Sections[i].Heading + "\n" + doc.Sections[i].Content)
		doc.Sections[i].References = normref.Extract(text, shortTitle)
	}

	doc.ShortTitle = shortTitle
	if strings.TrimSpace(doc.Jurisdiction) == "" {
		doc.Jurisdiction = strings.TrimSpace(p.Jurisdiction)
	}
	if doc.Type == "" {
		doc.Type = godr.DocumentTypeLaw
	}

	return doc, nil
}

func detectShortTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	m := shortTitlePattern.FindStringSubmatch(title)
	if len(m) < 2 {
		return ""
	}
	abbr := strings.TrimSpace(m[1])
	abbr = strings.Join(strings.Fields(abbr), " ")
	if len(abbr) > 24 {
		return ""
	}
	return abbr
}
