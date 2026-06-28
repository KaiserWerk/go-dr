package juris

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/PuerkitoBio/goquery"
)

const defaultListSelector = `a[href*="jportal"], a[href*="doc.id"], a[href*="doc.part"], a[href*="quelle=jlink"], a[href*="quelle.jlink"]`

func parseListDocumentRefs(baseURL string, cfg Config, raw []byte) ([]godr.DocumentRef, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}

	selector := cfg.ListSelector
	if strings.TrimSpace(selector) == "" {
		selector = defaultListSelector
	}

	seen := make(map[string]struct{})
	refs := make([]godr.DocumentRef, 0)

	doc.Find(selector).Each(func(i int, sel *goquery.Selection) {
		href, ok := sel.Attr("href")
		if !ok {
			return
		}

		resolved := resolveURL(baseURL, href)
		resolved = normalizeDocURL(resolved)
		if resolved == "" || !isLikelyJurisDocumentURL(resolved, cfg.BaseURL, cfg.AllowedHosts) {
			return
		}
		if _, ok := seen[resolved]; ok {
			return
		}
		seen[resolved] = struct{}{}

		title := extractListTitle(sel)
		if title == "" {
			title = guessTitleFromContext(sel)
		}

		id := deriveJurisID(resolved)
		if id == "" {
			id = fmt.Sprintf("%s-%d", strings.ToLower(strings.TrimSpace(cfg.Name)), len(refs)+1)
		}

		refs = append(refs, godr.DocumentRef{
			ID:  id,
			URL: resolved,
			Metadata: map[string]string{
				"title": title,
			},
		})
	})

	return refs, nil
}

func extractListTitle(sel *goquery.Selection) string {
	if sel == nil {
		return ""
	}
	if title := compactSpaces(strings.TrimSpace(sel.Text())); title != "" {
		return title
	}
	for _, attr := range []string{"title", "aria-label", "data-title"} {
		if v, ok := sel.Attr(attr); ok {
			v = compactSpaces(strings.TrimSpace(v))
			if v != "" {
				return v
			}
		}
	}
	return ""
}

func guessTitleFromContext(sel *goquery.Selection) string {
	if sel == nil {
		return ""
	}
	if tr := sel.Closest("tr"); tr != nil && tr.Length() > 0 {
		if row := compactSpaces(strings.TrimSpace(tr.Text())); row != "" {
			return trimTo(row, 260)
		}
	}
	if li := sel.Closest("li"); li != nil && li.Length() > 0 {
		if item := compactSpaces(strings.TrimSpace(li.Text())); item != "" {
			return trimTo(item, 260)
		}
	}
	return ""
}

func normalizeDocURL(v string) string {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil {
		return ""
	}
	u.Fragment = ""
	return u.String()
}

func deriveJurisID(v string) string {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil {
		return ""
	}

	q := u.Query()
	for _, key := range []string{"doc.id", "doknr", "id", "j"} {
		if val := strings.TrimSpace(q.Get(key)); val != "" {
			return strings.ReplaceAll(key, ".", "_") + "-" + val
		}
	}

	name := path.Base(u.Path)
	name = strings.TrimSpace(strings.Trim(name, "/"))
	if name == "" || name == "." {
		return ""
	}
	return name
}

func compactSpaces(v string) string {
	return strings.Join(strings.Fields(v), " ")
}

func trimTo(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return strings.TrimSpace(v[:max])
}
