package normref

import (
	"regexp"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
)

var chainConnectorPattern = regexp.MustCompile(`(?i)\b(?:i\s*\.\s*v\s*\.\s*m\s*\.|ivm|in\s+verbindung\s+mit)\b`)

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
	for i := 0; i < len(hits)-1; i++ {
		left := hits[i]
		right := hits[i+1]
		if left.end > len(text) || right.start < 0 || right.start > len(text) || left.end > right.start {
			continue
		}

		between := text[left.end:right.start]
		if !chainConnectorPattern.MatchString(between) {
			continue
		}

		if left.target == "" || right.target == "" || left.target == right.target {
			continue
		}

		key := left.target + "->" + right.target
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, godr.NormChain{Source: left.target, Target: right.target})
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
