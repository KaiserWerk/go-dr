package juris

import (
	"net/url"
	"sort"
	"strings"
)

const (
	ProfileBadenWuerttemberg     = "baden-wuerttemberg"
	ProfileBerlin                = "berlin"
	ProfileHessen                = "hessen"
	ProfileMecklenburgVorpommern = "mecklenburg-vorpommern"
	ProfileRheinlandPfalz        = "rheinland-pfalz"
	ProfileSachsenAnhalt         = "sachsen-anhalt"
	ProfileSchleswigHolstein     = "schleswig-holstein"
	ProfileThueringen            = "thueringen"
)

// profile contains tuned entry points and selectors for Juris-based portals.
type profile struct {
	baseURL      string
	listPath     string
	listQuery    map[string]string
	listURL      string
	listSelector string
	allowedHosts []string
}

var portalProfiles = map[string]profile{
	ProfileBadenWuerttemberg: {
		baseURL:      "https://www.landesrecht-bw.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht-bw.de", "landesrecht-bw.de"},
	},
	ProfileBerlin: {
		baseURL:      "https://gesetze.berlin.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"gesetze.berlin.de"},
	},
	ProfileHessen: {
		baseURL:      "https://www.rv.hessenrecht.hessen.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.rv.hessenrecht.hessen.de", "rv.hessenrecht.hessen.de"},
	},
	ProfileMecklenburgVorpommern: {
		baseURL:      "https://www.landesrecht-mv.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht-mv.de", "landesrecht-mv.de"},
	},
	ProfileRheinlandPfalz: {
		baseURL:      "https://landesrecht.rlp.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"landesrecht.rlp.de"},
	},
	ProfileSachsenAnhalt: {
		baseURL:      "https://www.landesrecht.sachsen-anhalt.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht.sachsen-anhalt.de", "landesrecht.sachsen-anhalt.de"},
	},
	ProfileSchleswigHolstein: {
		baseURL:      "https://www.gesetze-rechtsprechung.sh.juris.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.gesetze-rechtsprechung.sh.juris.de", "gesetze-rechtsprechung.sh.juris.de"},
	},
	ProfileThueringen: {
		baseURL:      "https://landesrecht.thueringen.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"landesrecht.thueringen.de"},
	},
}

var profileJurisdiction = map[string]string{
	ProfileBadenWuerttemberg:     "DE-BW",
	ProfileBerlin:                "DE-BE",
	ProfileHessen:                "DE-HE",
	ProfileMecklenburgVorpommern: "DE-MV",
	ProfileRheinlandPfalz:        "DE-RP",
	ProfileSachsenAnhalt:         "DE-ST",
	ProfileSchleswigHolstein:     "DE-SH",
	ProfileThueringen:            "DE-TH",
}

func applyProfile(cfg Config) Config {
	name := strings.TrimSpace(cfg.Name)
	p, ok := portalProfiles[name]
	if !ok {
		return cfg
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = p.baseURL
	}
	if cfg.ListPath == "" {
		cfg.ListPath = p.listPath
	}
	if len(cfg.ListQuery) == 0 && len(p.listQuery) > 0 {
		cfg.ListQuery = copyStringMap(p.listQuery)
	}
	if cfg.ListURL == "" {
		cfg.ListURL = p.listURL
	}
	if cfg.ListSelector == "" {
		cfg.ListSelector = p.listSelector
	}
	if len(cfg.AllowedHosts) == 0 {
		cfg.AllowedHosts = append([]string(nil), p.allowedHosts...)
	}
	if cfg.ListURL == "" {
		cfg.ListURL = buildListURL(cfg.BaseURL, cfg.ListPath, cfg.ListQuery)
	}
	if cfg.Jurisdiction == "" {
		cfg.Jurisdiction = profileJurisdiction[name]
	}
	return cfg
}

// ProfileNames returns all known profile identifiers in stable order.
func ProfileNames() []string {
	names := make([]string, 0, len(portalProfiles))
	for name := range portalProfiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// NewProfileSource returns a source with profile defaults and jurisdiction.
func NewProfileSource(profileName string) Source {
	profileName = strings.TrimSpace(profileName)
	return NewSource(Config{Name: profileName, Jurisdiction: profileJurisdiction[profileName]})
}

// NewProfileSourceWithQuery returns a profile source and applies query overrides.
func NewProfileSourceWithQuery(profileName string, overrides map[string]string) Source {
	return NewProfileSource(profileName).WithListQuery(overrides)
}

// BootstrapListQuery returns the profile default query map for a given state profile name.
func BootstrapListQuery(profileName string) map[string]string {
	p, ok := portalProfiles[strings.TrimSpace(profileName)]
	if !ok {
		return nil
	}
	return copyStringMap(p.listQuery)
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func buildURL(baseURL, relPath string, query map[string]string) string {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return strings.TrimSpace(baseURL)
	}

	if strings.TrimSpace(relPath) != "" {
		rel, err := url.Parse(strings.TrimSpace(relPath))
		if err == nil {
			base = base.ResolveReference(rel)
		}
	}

	q := base.Query()
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := strings.TrimSpace(query[k])
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	base.RawQuery = q.Encode()
	base.Fragment = ""
	return base.String()
}
