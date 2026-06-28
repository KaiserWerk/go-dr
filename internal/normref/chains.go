package normref

import (
	"regexp"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
)

var (
	strongChainConnectorPattern = regexp.MustCompile(`(?i)(?:\bi\s*\.?\s*v\s*\.?\s*m\s*\.?\b|\bivm\b|\bin\s+verbindung\s+mit\b|\bin\s+verbindung\s+zu\b|\bgemae(?:ss|ß)\b|\bgem\.\b|\bnach\s+maßgabe\s+von\b|\bnach\s+massgabe\s+von\b|\bentsprechend\b)`)
	weakListConnectorPattern    = regexp.MustCompile(`(?i)^\s*(?:,|und|oder|sowie|bzw\.?)\s*$`)
	sentenceBoundaryPattern     = regexp.MustCompile(`[.!?;\n\r]`)
)

type refHit struct {
	start  int
	end    int
	target string
}

// ExtractChains returns directed reference chains (source -> target) found in text.
func ExtractChains(text, defaultLaw string) []godr.NormChain {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	defaultLaw = normalizeLawToken(defaultLaw)
	lastLaw := ""
	hits := make([]refHit, 0)

	for _, idx := range singleRefPattern.FindAllStringSubmatchIndex(text, -1) {
		if len(idx) < 6 {
			continue
		}

		kindStart, kindEnd := idx[2], idx[3]
		numStart, numEnd := idx[4], idx[5]
		if kindStart < 0 || numStart < 0 {
			continue
		}

		kind := strings.TrimSpace(text[kindStart:kindEnd])
		num := strings.TrimSpace(text[numStart:numEnd])
		if kind == "" || num == "" {
			continue
		}

		law, explicit := detectLawToken(text, idx[0], idx[1], lastLaw, defaultLaw)
		if explicit {
			lastLaw = law
		}

		token := buildRefToken(kind, num, idx, text)
		if token == "" {
			continue
		}

		hits = append(hits, refHit{
			start:  idx[0],
			end:    idx[1],
			target: strings.TrimSpace(law + " " + token),
		})
	}

	if len(hits) < 2 {
		return nil
	}

	out := make([]godr.NormChain, 0)
	seen := make(map[string]struct{})
	carrySource := ""
	for i := 0; i < len(hits)-1; i++ {
		left := hits[i]
		right := hits[i+1]
		if left.end > len(text) || right.start < 0 || right.start > len(text) || left.end > right.start {
			carrySource = ""
			continue
		}

		between := text[left.end:right.start]

		if isStrongChainConnector(between) {
			appendChain(&out, seen, left.target, right.target)
			carrySource = left.target
			continue
		}

		if carrySource != "" && isWeakChainContinuation(between) {
			appendChain(&out, seen, carrySource, right.target)
			continue
		}

		if sentenceBoundaryPattern.MatchString(between) || len(strings.TrimSpace(between)) > 120 {
			carrySource = ""
			continue
		}

		carrySource = ""
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func isStrongChainConnector(between string) bool {
	between = strings.TrimSpace(between)
	if between == "" {
		return false
	}
	if len(between) > 120 {
		return false
	}
	if strings.Contains(between, "§") || strings.Contains(strings.ToLower(between), "art") {
		return false
	}
	if strongChainConnectorPattern.MatchString(between) {
		return true
	}

	norm := normalizeConnectorWindow(between)
	switch norm {
	case "ivm", "inverbindungmit", "inverbindungzu", "gemaess", "gemass", "gem", "nachmassgabevon", "entsprechend":
		return true
	default:
		return false
	}
}

func isWeakChainContinuation(between string) bool {
	between = strings.TrimSpace(between)
	if between == "" {
		return false
	}
	if len(between) > 40 {
		return false
	}
	if sentenceBoundaryPattern.MatchString(between) {
		return false
	}
	return weakListConnectorPattern.MatchString(between)
}

func normalizeConnectorWindow(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	replacer := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		".", "",
		",", "",
		":", "",
		"-", "",
		"_", "",
		"(", "",
		")", "",
	)
	return replacer.Replace(v)
}

func appendChain(out *[]godr.NormChain, seen map[string]struct{}, source, target string) {
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if source == "" || target == "" || source == target {
		return
	}
	key := source + "->" + target
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*out = append(*out, godr.NormChain{Source: source, Target: target})
}
