package juris

import "strings"

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
	"baden-wuerttemberg": {
		baseURL:      "https://www.landesrecht-bw.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht-bw.de", "landesrecht-bw.de"},
	},
	"berlin": {
		baseURL:      "https://gesetze.berlin.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"gesetze.berlin.de"},
	},
	"hessen": {
		baseURL:      "https://www.rv.hessenrecht.hessen.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.rv.hessenrecht.hessen.de", "rv.hessenrecht.hessen.de"},
	},
	"mecklenburg-vorpommern": {
		baseURL:      "https://www.landesrecht-mv.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht-mv.de", "landesrecht-mv.de"},
	},
	"rheinland-pfalz": {
		baseURL:      "https://landesrecht.rlp.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"landesrecht.rlp.de"},
	},
	"sachsen-anhalt": {
		baseURL:      "https://www.landesrecht.sachsen-anhalt.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht.sachsen-anhalt.de", "landesrecht.sachsen-anhalt.de"},
	},
	"schleswig-holstein": {
		baseURL:      "https://www.gesetze-rechtsprechung.sh.juris.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.gesetze-rechtsprechung.sh.juris.de", "gesetze-rechtsprechung.sh.juris.de"},
	},
	"thueringen": {
		baseURL:      "https://landesrecht.thueringen.de/",
		listPath:     "/jportal/",
		listQuery:    map[string]string{"quelle": "jlink", "query": "*", "showdoccase": "1"},
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"landesrecht.thueringen.de"},
	},
}

func applyProfile(cfg Config) Config {
	p, ok := portalProfiles[cfg.Name]
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
	return cfg
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
