package nrw

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/crawler"
	"github.com/PuerkitoBio/goquery"
)

const (
	// DefaultListURL is a practical entry page for NRW legal search/listing.
	DefaultListURL = "https://recht.nrw.de/"
)

// Source implements NRW ingestion from recht.nrw.de.
type Source struct {
	Client       HTTPClient
	Parser       Parser
	ListURL      string
	ListSelector string
}

// HTTPClient is the minimal dependency required from a fetch client.
type HTTPClient interface {
	Get(ctx context.Context, url string) (int, []byte, error)
}

// Name implements godr.Source.
func (s Source) Name() string {
	return "nrw"
}

// ListDocuments parses a listing page and extracts document links.
func (s Source) ListDocuments(ctx context.Context) ([]godr.DocumentRef, error) {
	client := s.ensureClient()
	listURL := s.listURL()

	status, payload, err := client.Get(ctx, listURL)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list request failed with status %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}

	selector := s.listSelector()
	seen := make(map[string]struct{})
	refs := make([]godr.DocumentRef, 0)

	doc.Find(selector).Each(func(i int, sel *goquery.Selection) {
		href, ok := sel.Attr("href")
		if !ok {
			return
		}
		resolved := resolveURL(listURL, href)
		if resolved == "" {
			return
		}
		if !isLikelyNRWDocumentURL(resolved) {
			return
		}
		if _, ok := seen[resolved]; ok {
			return
		}
		seen[resolved] = struct{}{}

		title := strings.TrimSpace(sel.Text())
		id := deriveIDFromURL(resolved)
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

// FetchDocument downloads one NRW document page and parses it into a LegalDocument.
func (s Source) FetchDocument(ctx context.Context, ref godr.DocumentRef) (*godr.LegalDocument, error) {
	if strings.TrimSpace(ref.URL) == "" {
		return nil, errors.New("document ref requires URL")
	}

	client := s.ensureClient()
	parser := s.ensureParser()

	status, payload, err := client.Get(ctx, ref.URL)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("document request failed with status %d", status)
	}

	doc, err := parser.Parse(payload)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("parser returned nil document")
	}

	if strings.TrimSpace(doc.ID) == "" {
		doc.ID = strings.TrimSpace(ref.ID)
	}
	if strings.TrimSpace(doc.Title) == "" && ref.Metadata != nil {
		doc.Title = strings.TrimSpace(ref.Metadata["title"])
	}
	doc.SourceURL = ref.URL
	if strings.TrimSpace(doc.Jurisdiction) == "" {
		doc.Jurisdiction = "DE-NW"
	}
	if doc.Type == "" {
		doc.Type = godr.DocumentTypeLaw
	}

	return doc, nil
}

func (s Source) listURL() string {
	if strings.TrimSpace(s.ListURL) != "" {
		return s.ListURL
	}
	return DefaultListURL
}

func (s Source) listSelector() string {
	if strings.TrimSpace(s.ListSelector) != "" {
		return s.ListSelector
	}
	return "a[href]"
}

func (s Source) ensureClient() HTTPClient {
	if s.Client != nil {
		return s.Client
	}
	return crawler.NewClient(crawler.Config{
		UserAgent: "go-dr/0.1",
		Retry: crawler.RetryPolicy{
			MaxAttempts: 3,
		},
	})
}

func (s Source) ensureParser() Parser {
	if s.Parser != nil {
		return s.Parser
	}
	return HTMLDocumentParser{}
}

func resolveURL(baseURL, href string) string {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

func deriveIDFromURL(v string) string {
	u, err := url.Parse(v)
	if err != nil {
		return ""
	}
	name := path.Base(u.Path)
	name = strings.TrimSpace(strings.Trim(name, "/"))
	if name == "" || name == "." {
		return ""
	}
	return name
}

func isLikelyNRWDocumentURL(v string) bool {
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "recht.nrw.de") {
		return false
	}
	pathLower := strings.ToLower(u.Path)
	queryLower := strings.ToLower(u.RawQuery)
	return strings.Contains(pathLower, "/lmi/") || strings.Contains(queryLower, "sg=") || strings.Contains(queryLower, "menu")
}
