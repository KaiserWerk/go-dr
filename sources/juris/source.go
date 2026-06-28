package juris

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/crawler"
)

// Config defines one Juris-based state portal connection.
type Config struct {
	Name         string
	BaseURL      string
	ListURL      string
	ListPath     string
	ListQuery    map[string]string
	ListSelector string
	AllowedHosts []string
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
	cfg = applyProfile(cfg)
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://www.landesrecht.example/"
	}
	if strings.TrimSpace(cfg.ListURL) == "" {
		cfg.ListURL = buildListURL(cfg.BaseURL, cfg.ListPath, cfg.ListQuery)
	}
	if strings.TrimSpace(cfg.ListSelector) == "" {
		cfg.ListSelector = defaultListSelector
	}
	if len(cfg.AllowedHosts) == 0 {
		if u, err := url.Parse(cfg.BaseURL); err == nil && strings.TrimSpace(u.Host) != "" {
			cfg.AllowedHosts = []string{strings.ToLower(strings.TrimSpace(u.Host))}
		}
	}
	return Source{cfg: cfg}
}

// Config returns a copy of the source config.
func (s Source) Config() Config {
	cfg := s.cfg
	cfg.ListQuery = cloneStringMap(s.cfg.ListQuery)
	cfg.AllowedHosts = append([]string(nil), s.cfg.AllowedHosts...)
	return cfg
}

// WithListQuery returns a new source with merged query overrides and rebuilt list URL.
func (s Source) WithListQuery(overrides map[string]string) Source {
	cfg := s.Config()
	if cfg.ListQuery == nil {
		cfg.ListQuery = map[string]string{}
	}
	for k, v := range overrides {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if strings.TrimSpace(v) == "" {
			delete(cfg.ListQuery, k)
			continue
		}
		cfg.ListQuery[k] = v
	}
	cfg.ListURL = buildListURL(cfg.BaseURL, cfg.ListPath, cfg.ListQuery)
	return NewSource(cfg)
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

	return parseListDocumentRefs(listURL, s.cfg, payload)
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

func buildListURL(baseURL, listPath string, query map[string]string) string {
	return buildURL(baseURL, listPath, query)
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isLikelyJurisDocumentURL(v, base string, allowedHosts []string) bool {
	u, err := url.Parse(v)
	if err != nil {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(u.Host))
	if len(allowedHosts) > 0 {
		ok := false
		for _, allowed := range allowedHosts {
			allowed = strings.ToLower(strings.TrimSpace(allowed))
			if allowed == "" {
				continue
			}
			if host == allowed {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	baseURL, err := url.Parse(base)
	if err == nil {
		if baseURL.Host != "" && !strings.EqualFold(baseURL.Host, u.Host) {
			if len(allowedHosts) == 0 {
				return false
			}
		}
	}

	pathLower := strings.ToLower(u.Path)
	queryLower := strings.ToLower(u.RawQuery)
	return strings.Contains(pathLower, "jportal") || strings.Contains(queryLower, "doc.id") || strings.Contains(queryLower, "doc.part") || strings.Contains(pathLower, "quelle.jlink")
}

// Preconfigured Juris-based source constructors.
func BadenWuerttemberg() Source {
	return NewSource(Config{
		Name:         ProfileBadenWuerttemberg,
		Jurisdiction: "DE-BW",
	})
}

func Berlin() Source {
	return NewSource(Config{
		Name:         ProfileBerlin,
		Jurisdiction: "DE-BE",
	})
}

func Hessen() Source {
	return NewSource(Config{
		Name:         ProfileHessen,
		Jurisdiction: "DE-HE",
	})
}

func MecklenburgVorpommern() Source {
	return NewSource(Config{
		Name:         ProfileMecklenburgVorpommern,
		Jurisdiction: "DE-MV",
	})
}

func RheinlandPfalz() Source {
	return NewSource(Config{
		Name:         ProfileRheinlandPfalz,
		Jurisdiction: "DE-RP",
	})
}

func SachsenAnhalt() Source {
	return NewSource(Config{
		Name:         ProfileSachsenAnhalt,
		Jurisdiction: "DE-ST",
	})
}

func SchleswigHolstein() Source {
	return NewSource(Config{
		Name:         ProfileSchleswigHolstein,
		Jurisdiction: "DE-SH",
	})
}

func Thueringen() Source {
	return NewSource(Config{
		Name:         ProfileThueringen,
		Jurisdiction: "DE-TH",
	})
}

// Convenience constructors with query overrides.
func BadenWuerttembergWithQuery(overrides map[string]string) Source {
	return BadenWuerttemberg().WithListQuery(overrides)
}

func BerlinWithQuery(overrides map[string]string) Source {
	return Berlin().WithListQuery(overrides)
}

func HessenWithQuery(overrides map[string]string) Source {
	return Hessen().WithListQuery(overrides)
}

func MecklenburgVorpommernWithQuery(overrides map[string]string) Source {
	return MecklenburgVorpommern().WithListQuery(overrides)
}

func RheinlandPfalzWithQuery(overrides map[string]string) Source {
	return RheinlandPfalz().WithListQuery(overrides)
}

func SachsenAnhaltWithQuery(overrides map[string]string) Source {
	return SachsenAnhalt().WithListQuery(overrides)
}

func SchleswigHolsteinWithQuery(overrides map[string]string) Source {
	return SchleswigHolstein().WithListQuery(overrides)
}

func ThueringenWithQuery(overrides map[string]string) Source {
	return Thueringen().WithListQuery(overrides)
}
