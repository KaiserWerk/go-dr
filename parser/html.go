package parser

import (
	"errors"
	"regexp"
	"strings"
	"time"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/PuerkitoBio/goquery"
)

// HTMLDocumentParser parses generic legal HTML into a LegalDocument.
type HTMLDocumentParser struct{}

var (
	effectiveFromPattern = regexp.MustCompile(`(?i)(?:inkraft(?:treten)?|gueltig\s+ab|gültig\s+ab|wirksam\s+ab|effective\s+from)\s*[:\-]?\s*([^;,.\n\r]+)`)
	effectiveToPattern   = regexp.MustCompile(`(?i)(?:ausserkraft(?:treten)?|außerkraft(?:treten)?|gueltig\s+bis|gültig\s+bis|wirksam\s+bis|effective\s+to|valid\s+to)\s*[:\-]?\s*([^;,.\n\r]+)`)
	publishedPattern     = regexp.MustCompile(`(?i)(?:verkuendet|verkündet|veroeffentlicht|veröffentlicht|ausfertigung|stand\s+vom|published|date)\s*[:\-]?\s*([^;,.\n\r]+)`)
)

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

	metaPublished, metaEffFrom, metaEffTo := extractMetaDates(docSel)
	bodyPublished, bodyEffFrom, bodyEffTo := extractBodyDates(docSel)
	publishedAt := firstDate(metaPublished, bodyPublished)
	effectiveFrom := firstDate(metaEffFrom, bodyEffFrom)
	effectiveTo := firstDate(metaEffTo, bodyEffTo)

	return &godr.LegalDocument{
		Title:         title,
		Type:          godr.DocumentTypeLaw,
		PublishedAt:   publishedAt,
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   effectiveTo,
		Sections:      sections,
	}, nil
}

func extractMetaDates(docSel *goquery.Document) (*time.Time, *time.Time, *time.Time) {
	if docSel == nil {
		return nil, nil, nil
	}

	metaVals := make([]string, 0, 12)
	docSel.Find("meta").Each(func(i int, m *goquery.Selection) {
		name := strings.ToLower(strings.TrimSpace(m.AttrOr("name", "")))
		prop := strings.ToLower(strings.TrimSpace(m.AttrOr("property", "")))
		itemProp := strings.ToLower(strings.TrimSpace(m.AttrOr("itemprop", "")))
		content := strings.TrimSpace(m.AttrOr("content", ""))
		if content == "" {
			return
		}

		key := name + "|" + prop + "|" + itemProp
		if strings.Contains(key, "date") || strings.Contains(key, "publish") || strings.Contains(key, "inkraft") || strings.Contains(key, "valid") || strings.Contains(key, "gueltig") || strings.Contains(key, "gültig") {
			metaVals = append(metaVals, content)
		}
	})

	var publishedAt *time.Time
	var effectiveFrom *time.Time
	var effectiveTo *time.Time
	for _, v := range metaVals {
		l := strings.ToLower(v)
		switch {
		case strings.Contains(l, "inkraft") || strings.Contains(l, "effectivefrom") || strings.Contains(l, "validfrom") || strings.Contains(l, "gültig ab") || strings.Contains(l, "gueltig ab"):
			if effectiveFrom == nil {
				effectiveFrom = parseDateLoose(v)
			}
		case strings.Contains(l, "ausserkraft") || strings.Contains(l, "außerkraft") || strings.Contains(l, "effectiveto") || strings.Contains(l, "validto") || strings.Contains(l, "gültig bis") || strings.Contains(l, "gueltig bis"):
			if effectiveTo == nil {
				effectiveTo = parseDateLoose(v)
			}
		default:
			if publishedAt == nil {
				publishedAt = parseDateLoose(v)
			}
		}
	}

	if publishedAt == nil && len(metaVals) > 0 {
		publishedAt = parseDateLoose(metaVals[0])
	}
	return publishedAt, effectiveFrom, effectiveTo
}

func extractBodyDates(docSel *goquery.Document) (*time.Time, *time.Time, *time.Time) {
	if docSel == nil {
		return nil, nil, nil
	}

	body := strings.TrimSpace(docSel.Find("body").Text())
	body = compactSpaces(body)
	if body == "" {
		return nil, nil, nil
	}

	var publishedAt *time.Time
	var effectiveFrom *time.Time
	var effectiveTo *time.Time

	if m := effectiveFromPattern.FindStringSubmatch(body); len(m) > 1 {
		effectiveFrom = parseDateLoose(m[1])
	}
	if m := effectiveToPattern.FindStringSubmatch(body); len(m) > 1 {
		effectiveTo = parseDateLoose(m[1])
	}
	if m := publishedPattern.FindStringSubmatch(body); len(m) > 1 {
		publishedAt = parseDateLoose(m[1])
	}

	if publishedAt == nil {
		publishedAt = parseDateLoose(body)
	}
	return publishedAt, effectiveFrom, effectiveTo
}

func firstDate(values ...*time.Time) *time.Time {
	for _, v := range values {
		if v != nil {
			copyV := *v
			return &copyV
		}
	}
	return nil
}

func compactSpaces(v string) string {
	return strings.Join(strings.Fields(v), " ")
}
