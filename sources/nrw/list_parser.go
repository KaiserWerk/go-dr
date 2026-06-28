package nrw

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/PuerkitoBio/goquery"
)

const defaultListSelector = `a[href*="/lmi/owa/br_"], a[href*="/lmi/owa"], a[href*="v_id="], a[href*="sg="], a[href*="menu="]`

func parseListDocumentRefs(baseURL string, raw []byte, selector string) ([]godr.DocumentRef, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}

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
		if resolved == "" || !isLikelyNRWDocumentURL(resolved) {
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

		id := deriveNRWID(resolved)
		if id == "" {
			id = fmt.Sprintf("nrw-%d", len(refs)+1)
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
	title := strings.TrimSpace(sel.Text())
	if title != "" {
		return compactSpaces(title)
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
		rowText := compactSpaces(strings.TrimSpace(tr.Text()))
		if rowText != "" {
			return trimTo(rowText, 240)
		}
	}
	if li := sel.Closest("li"); li != nil && li.Length() > 0 {
		itemText := compactSpaces(strings.TrimSpace(li.Text()))
		if itemText != "" {
			return trimTo(itemText, 240)
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

func deriveNRWID(v string) string {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil {
		return ""
	}

	q := u.Query()
	for _, key := range []string{"v_id", "id", "docid", "doc.id", "j"} {
		if val := strings.TrimSpace(q.Get(key)); val != "" {
			return key + "-" + val
		}
	}

	name := path.Base(u.Path)
	name = strings.TrimSpace(strings.Trim(name, "/"))
	if name != "" && name != "." {
		return name
	}
	return ""
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
