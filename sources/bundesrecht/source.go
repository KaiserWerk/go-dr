package bundesrecht

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/crawler"
)

const (
	// DefaultTOCURL points to the official Bundesrecht TOC index.
	DefaultTOCURL = "https://www.gesetze-im-internet.de/gii-toc.xml"
	defaultLawURL = "https://www.gesetze-im-internet.de/%s/xml.zip"
)

// Source implements Bundesrecht ingestion from gesetze-im-internet.de.
type Source struct {
	Client HTTPClient
	Parser Parser
	TOCURL string
}

// HTTPClient is the minimal dependency required from a fetch client.
type HTTPClient interface {
	Get(ctx context.Context, url string) (int, []byte, error)
}

// Name implements godr.Source.
func (s Source) Name() string {
	return "bundesrecht"
}

// ListDocuments reads gii-toc.xml and maps entries into DocumentRef values.
func (s Source) ListDocuments(ctx context.Context) ([]godr.DocumentRef, error) {
	client := s.ensureClient()
	status, payload, err := client.Get(ctx, s.tocURL())
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("toc request failed with status %d", status)
	}

	items, err := parseTOC(payload)
	if err != nil {
		return nil, err
	}

	refs := make([]godr.DocumentRef, 0, len(items))
	for _, item := range items {
		id := deriveIDFromURL(item.Link)
		if id == "" {
			continue
		}
		refs = append(refs, godr.DocumentRef{
			ID:  id,
			URL: normalizeHTTP(item.Link),
			Metadata: map[string]string{
				"title": strings.TrimSpace(item.Title),
			},
		})
	}

	return refs, nil
}

// FetchDocument downloads one XML ZIP and parses it into a LegalDocument.
func (s Source) FetchDocument(ctx context.Context, ref godr.DocumentRef) (*godr.LegalDocument, error) {
	client := s.ensureClient()
	parser := s.ensureParser()

	url := strings.TrimSpace(ref.URL)
	if url == "" {
		if strings.TrimSpace(ref.ID) == "" {
			return nil, errors.New("document ref requires ID or URL")
		}
		url = fmt.Sprintf(defaultLawURL, ref.ID)
	}

	status, payload, err := client.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("document request failed with status %d", status)
	}

	rawXML, err := firstXMLFromZIP(payload)
	if err != nil {
		return nil, err
	}

	doc, err := parser.Parse(rawXML)
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
	doc.SourceURL = url
	if strings.TrimSpace(doc.Jurisdiction) == "" {
		doc.Jurisdiction = "DE-BUND"
	}
	if doc.Type == "" {
		doc.Type = godr.DocumentTypeLaw
	}

	return doc, nil
}

type toc struct {
	Items []tocItem `xml:"item"`
}

type tocItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

func parseTOC(raw []byte) ([]tocItem, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty toc payload")
	}
	var doc toc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return doc.Items, nil
}

func firstXMLFromZIP(rawZIP []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(rawZIP), int64(len(rawZIP)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(rc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	return nil, errors.New("no XML file in zip archive")
}

func deriveIDFromURL(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = normalizeHTTP(v)
	name := path.Base(path.Dir(v))
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func normalizeHTTP(v string) string {
	if strings.HasPrefix(v, "http://") {
		return "https://" + strings.TrimPrefix(v, "http://")
	}
	return v
}

func (s Source) tocURL() string {
	if strings.TrimSpace(s.TOCURL) != "" {
		return s.TOCURL
	}
	return DefaultTOCURL
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
	return XMLDocumentParser{}
}
