package juris

// profile contains tuned entry points and selectors for Juris-based portals.
type profile struct {
	baseURL      string
	listURL      string
	listSelector string
	allowedHosts []string
}

var portalProfiles = map[string]profile{
	"baden-wuerttemberg": {
		baseURL:      "https://www.landesrecht-bw.de/",
		listURL:      "https://www.landesrecht-bw.de/jportal/",
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht-bw.de", "landesrecht-bw.de"},
	},
	"berlin": {
		baseURL:      "https://gesetze.berlin.de/",
		listURL:      "https://gesetze.berlin.de/jportal/",
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"gesetze.berlin.de"},
	},
	"hessen": {
		baseURL:      "https://www.rv.hessenrecht.hessen.de/",
		listURL:      "https://www.rv.hessenrecht.hessen.de/jportal/",
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.rv.hessenrecht.hessen.de", "rv.hessenrecht.hessen.de"},
	},
	"mecklenburg-vorpommern": {
		baseURL:      "https://www.landesrecht-mv.de/",
		listURL:      "https://www.landesrecht-mv.de/jportal/",
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht-mv.de", "landesrecht-mv.de"},
	},
	"rheinland-pfalz": {
		baseURL:      "https://landesrecht.rlp.de/",
		listURL:      "https://landesrecht.rlp.de/jportal/",
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"landesrecht.rlp.de"},
	},
	"sachsen-anhalt": {
		baseURL:      "https://www.landesrecht.sachsen-anhalt.de/",
		listURL:      "https://www.landesrecht.sachsen-anhalt.de/jportal/",
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.landesrecht.sachsen-anhalt.de", "landesrecht.sachsen-anhalt.de"},
	},
	"schleswig-holstein": {
		baseURL:      "https://www.gesetze-rechtsprechung.sh.juris.de/",
		listURL:      "https://www.gesetze-rechtsprechung.sh.juris.de/jportal/",
		listSelector: `a[href*="quelle=jlink"], a[href*="doc.id"], a[href*="jportal"]`,
		allowedHosts: []string{"www.gesetze-rechtsprechung.sh.juris.de", "gesetze-rechtsprechung.sh.juris.de"},
	},
	"thueringen": {
		baseURL:      "https://landesrecht.thueringen.de/",
		listURL:      "https://landesrecht.thueringen.de/jportal/",
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
	if cfg.ListURL == "" {
		cfg.ListURL = p.listURL
	}
	if cfg.ListSelector == "" {
		cfg.ListSelector = p.listSelector
	}
	if len(cfg.AllowedHosts) == 0 {
		cfg.AllowedHosts = append([]string(nil), p.allowedHosts...)
	}
	return cfg
}
