package parser

import (
	"errors"
	"strings"

	"github.com/KaiserWerk/go-dr"
	"github.com/PuerkitoBio/goquery"
)

// HTMLDocumentParser parses generic legal HTML into a LegalDocument.
type HTMLDocumentParser struct{}

// Parse implements Parser.
func (p HTMLDocumentParser) Parse(raw []byte) (*godr.LegalDocument, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty HTML payload")
	}

	docSel, err := goquery.NewDocumentFromReader(strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(docSel.Find("h1").First().Text())
	if title == "" {
		title = strings.TrimSpace(docSel.Find("title").First().Text())
	}

	sections := make([]godr.Section, 0)
	docSel.Find("section").Each(func(i int, s *goquery.Selection) {
		heading := strings.TrimSpace(s.Find("h2, h3").First().Text())
		content := strings.TrimSpace(s.Find("p").First().Text())
		if heading == "" && content == "" {
			return
		}
		sections = append(sections, godr.Section{
			Identifier: strings.TrimSpace(s.AttrOr("id", "")),
			Heading:    heading,
			Content:    content,
		})
	})

	if len(sections) == 0 {
		fallback := strings.TrimSpace(docSel.Find("body").Text())
		if fallback != "" {
			sections = append(sections, godr.Section{Content: fallback})
		}
	}

	return &godr.LegalDocument{
		Title:    title,
		Type:     godr.DocumentTypeLaw,
		Sections: sections,
	}, nil
}
