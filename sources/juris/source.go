package juris

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

// Config defines one Juris-based state portal connection.
type Config struct {
	Name         string
	BaseURL      string
	ListURL      string
	ListSelector string
	Jurisdiction string
}

// Source is a shared adapter for Juris-based state portals.
type Source struct {
	cfg    Config
	Client HTTPClient
	Parser Parser
}

// HTTPClient is the minimal dependency required from a fetch client.
type HTTPClient interface {
	Get(ctx context.Context, url string) (int, []byte, error)
}

// NewSource creates a Juris source adapter with defaults applied.
func NewSource(cfg Config) Source {
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = "juris"
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://www.landesrecht.example/"
	}
	if strings.TrimSpace(cfg.ListURL) == "" {
		cfg.ListURL = cfg.BaseURL
	}
	if strings.TrimSpace(cfg.ListSelector) == "" {
		cfg.ListSelector = `a[href*="jportal"], a[href*="doc.id"], a[href*="doc.part"], a[href*="quelle.jlink"]`
	}
	return Source{cfg: cfg}
}

// Name implements godr.Source.
func (s Source) Name() string {
	return "juris-" + strings.ToLower(strings.TrimSpace(s.cfg.Name))
}

// ListDocuments parses one list page and extracts likely legal document links.
func (s Source) ListDocuments(ctx context.Context) ([]godr.DocumentRef, error) {
	client := s.ensureClient()
	listURL := s.cfg.ListURL

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

	selector := s.cfg.ListSelector
	seen := make(map[string]struct{})
	refs := make([]godr.DocumentRef, 0)

	doc.Find(selector).Each(func(i int, sel *goquery.Selection) {
		href, ok := sel.Attr("href")
		if !ok {
			return
		}
		resolved := resolveURL(listURL, href)
		if resolved == "" || !isLikelyJurisDocumentURL(resolved, s.cfg.BaseURL) {
			return
		}
		if _, ok := seen[resolved]; ok {
			return
		}
		seen[resolved] = struct{}{}

		title := strings.TrimSpace(strings.Join(strings.Fields(sel.Text()), " "))
		if title == "" {
			title = strings.TrimSpace(sel.AttrOr("title", ""))
		}

		id := deriveIDFromURL(resolved)
		if id == "" {
			id = fmt.Sprintf("%s-%d", strings.ToLower(strings.TrimSpace(s.cfg.Name)), len(refs)+1)
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

// FetchDocument downloads one Juris page and parses it into a LegalDocument.
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
		doc.Jurisdiction = strings.TrimSpace(s.cfg.Jurisdiction)
	}
	if doc.Type == "" {
		doc.Type = godr.DocumentTypeLaw
	}

	return doc, nil
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
	return HTMLDocumentParser{Jurisdiction: s.cfg.Jurisdiction}
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

func deriveIDFromURL(v string) string {
	u, err := url.Parse(v)
	if err != nil {
		return ""
	}

	q := u.Query()
	for _, key := range []string{"doc.id", "doknr", "id"} {
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

func isLikelyJurisDocumentURL(v, base string) bool {
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	baseURL, err := url.Parse(base)
	if err == nil {
		if baseURL.Host != "" && !strings.EqualFold(baseURL.Host, u.Host) {
			return false
		}
	}

	pathLower := strings.ToLower(u.Path)
	queryLower := strings.ToLower(u.RawQuery)
	return strings.Contains(pathLower, "jportal") || strings.Contains(queryLower, "doc.id") || strings.Contains(queryLower, "doc.part") || strings.Contains(pathLower, "quelle.jlink")
}

// Preconfigured Juris-based source constructors.
func BadenWuerttemberg() Source {
	return NewSource(Config{
		Name:         "baden-wuerttemberg",
		BaseURL:      "https://www.landesrecht-bw.de/",
		Jurisdiction: "DE-BW",
	})
}

func Berlin() Source {
	return NewSource(Config{
		Name:         "berlin",
		BaseURL:      "https://gesetze.berlin.de/",
		Jurisdiction: "DE-BE",
	})
}

func Hessen() Source {
	return NewSource(Config{
		Name:         "hessen",
		BaseURL:      "https://www.rv.hessenrecht.hessen.de/",
		Jurisdiction: "DE-HE",
	})
}

func MecklenburgVorpommern() Source {
	return NewSource(Config{
		Name:         "mecklenburg-vorpommern",
		BaseURL:      "https://www.landesrecht-mv.de/",
		Jurisdiction: "DE-MV",
	})
}

func RheinlandPfalz() Source {
	return NewSource(Config{
		Name:         "rheinland-pfalz",
		BaseURL:      "https://landesrecht.rlp.de/",
		Jurisdiction: "DE-RP",
	})
}

func SachsenAnhalt() Source {
	return NewSource(Config{
		Name:         "sachsen-anhalt",
		BaseURL:      "https://www.landesrecht.sachsen-anhalt.de/",
		Jurisdiction: "DE-ST",
	})
}

func SchleswigHolstein() Source {
	return NewSource(Config{
		Name:         "schleswig-holstein",
		BaseURL:      "https://www.gesetze-rechtsprechung.sh.juris.de/",
		Jurisdiction: "DE-SH",
	})
}

func Thueringen() Source {
	return NewSource(Config{
		Name:         "thueringen",
		BaseURL:      "https://landesrecht.thueringen.de/",
		Jurisdiction: "DE-TH",
	})
}
