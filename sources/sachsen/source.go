package sachsen

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/crawler"
)

const (
	// DefaultListURL is a practical entry page for REVOSax listing/search.
	DefaultListURL = "https://www.revosax.sachsen.de/"
)

// Source implements Sachsen ingestion from revosax.sachsen.de.
type Source struct {
	Client       HTTPClient
	Parser       Parser
	Snapshots    crawler.SnapshotStore
	ListURL      string
	ListSelector string
}

// HTTPClient is the minimal dependency required from a fetch client.
type HTTPClient interface {
	Get(ctx context.Context, url string) (int, []byte, error)
}

// Name implements godr.Source.
func (s Source) Name() string {
	return "sachsen"
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
	_ = s.saveSnapshot("list", payload)

	return parseListDocumentRefs(listURL, payload, s.listSelector())
}

// FetchDocument downloads one Sachsen document page and parses it into a LegalDocument.
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
	_ = s.saveSnapshot(ref.ID, payload)

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
		doc.Jurisdiction = "DE-SN"
	}
	if doc.Type == "" {
		doc.Type = godr.DocumentTypeLaw
	}
	doc.EnsureVersions()

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
	return defaultListSelector
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
	resolved := base.ResolveReference(u)
	resolved.Fragment = ""
	return resolved.String()
}

func isLikelySachsenDocumentURL(v string) bool {
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "revosax.sachsen.de") {
		return false
	}
	pathLower := strings.ToLower(u.Path)
	queryLower := strings.ToLower(u.RawQuery)
	return strings.Contains(pathLower, "vorschrift") || strings.Contains(pathLower, "dokument") || strings.Contains(pathLower, "revosax") || strings.Contains(queryLower, "id=")
}

func (s Source) saveSnapshot(documentID string, payload []byte) error {
	if s.Snapshots == nil || len(payload) == 0 {
		return nil
	}
	_, err := s.Snapshots.Save(s.Name(), documentID, payload)
	return err
}
