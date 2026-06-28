package normref

import (
	"fmt"
	"regexp"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
)

var (
	multiParagraphPattern = regexp.MustCompile(`(?i)§§\s*([0-9]+[a-zA-Z]*(?:\s*,\s*[0-9]+[a-zA-Z]*)*(?:\s*(?:und|oder)\s*[0-9]+[a-zA-Z]*)?)`)
	singleRefPattern      = regexp.MustCompile(`(?i)(§|Art\.?)\s*([0-9]+[a-zA-Z]*)(?:\s*Abs\.\s*([0-9]+[a-zA-Z]*))?(?:\s*S\.\s*([0-9]+))?`)
	suffixLawPattern      = regexp.MustCompile(`^\s*(?:[\),;:]\s*)?(?:i\.\s*V\.\s*m\.\s*)?([A-Z][A-Z0-9]{1,12})\b`)
	prefixLawPattern      = regexp.MustCompile(`([A-Z][A-Z0-9]{1,12})\s*$`)
	multiSplitPattern     = regexp.MustCompile(`\s*(?:,|und|oder)\s*`)
)

var lawPhraseAliases = map[string]string{
	"buergerliches gesetzbuch":   "BGB",
	"grundgesetz":                "GG",
	"strafgesetzbuch":            "StGB",
	"zivilprozessordnung":        "ZPO",
	"verwaltungsgerichtsordnung": "VwGO",
}

// Extract returns normalized legal references found in text.
func Extract(text, defaultLaw string) []godr.Reference {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	defaultLaw = normalizeLawToken(defaultLaw)
	seen := make(map[string]struct{})
	out := make([]godr.Reference, 0)
	lastLaw := ""

	for _, idx := range multiParagraphPattern.FindAllStringSubmatchIndex(text, -1) {
		if len(idx) < 4 {
			continue
		}
		rangeStart, rangeEnd := idx[0], idx[1]
		listStart, listEnd := idx[2], idx[3]
		if listStart < 0 || listEnd <= listStart {
			continue
		}
		law, explicit := detectLawToken(text, rangeStart, rangeEnd, lastLaw, defaultLaw)
		if explicit {
			lastLaw = law
			appendReference(&out, seen, godr.ReferenceTypeLaw, law)
		}

		for _, part := range multiSplitPattern.Split(text[listStart:listEnd], -1) {
			num := strings.TrimSpace(part)
			if num == "" {
				continue
			}
			target := fmt.Sprintf("%s § %s", law, num)
			appendReference(&out, seen, godr.ReferenceTypeParagraph, target)
		}
	}

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
		if num == "" {
			continue
		}

		law, explicit := detectLawToken(text, idx[0], idx[1], lastLaw, defaultLaw)
		if explicit {
			lastLaw = law
			appendReference(&out, seen, godr.ReferenceTypeLaw, law)
		}

		token := buildRefToken(kind, num, idx, text)
		if token == "" {
			continue
		}

		target := law + " " + token
		if strings.HasPrefix(strings.ToLower(kind), "art") {
			appendReference(&out, seen, godr.ReferenceTypeArticle, target)
			continue
		}
		appendReference(&out, seen, godr.ReferenceTypeParagraph, target)
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func buildRefToken(kind, number string, idx []int, text string) string {
	kind = strings.TrimSpace(kind)
	number = strings.TrimSpace(number)
	if kind == "" || number == "" {
		return ""
	}

	kind = normalizeRefKind(kind)
	token := fmt.Sprintf("%s %s", kind, number)

	if len(idx) > 6 && idx[6] >= 0 {
		abs := strings.TrimSpace(text[idx[6]:idx[7]])
		if abs != "" {
			token += " Abs. " + abs
		}
	}
	if len(idx) > 8 && idx[8] >= 0 {
		s := strings.TrimSpace(text[idx[8]:idx[9]])
		if s != "" {
			token += " S. " + s
		}
	}

	return token
}

func normalizeRefKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if strings.HasPrefix(strings.ToLower(kind), "art") {
		return "Art."
	}
	return "§"
}

func appendReference(out *[]godr.Reference, seen map[string]struct{}, typ godr.ReferenceType, target string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	key := string(typ) + ":" + target
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*out = append(*out, godr.Reference{Type: typ, Target: target})
}

func detectLawToken(text string, start, end int, lastLaw, defaultLaw string) (string, bool) {
	if law := normalizeLawToken(findSuffixLaw(text, end)); law != "" {
		return law, true
	}
	if law := normalizeLawToken(findPrefixLaw(text, start)); law != "" {
		return law, true
	}
	if law := normalizeLawToken(findLawFromPhrase(text, start, end)); law != "" {
		return law, true
	}
	if law := normalizeLawToken(lastLaw); law != "" {
		return law, false
	}
	if law := normalizeLawToken(defaultLaw); law != "" {
		return law, false
	}
	return "UNSPECIFIED", false
}

func findSuffixLaw(text string, end int) string {
	if end < 0 {
		end = 0
	}
	to := end + 48
	if to > len(text) {
		to = len(text)
	}
	window := text[end:to]
	m := suffixLawPattern.FindStringSubmatch(window)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func findPrefixLaw(text string, start int) string {
	if start < 0 {
		start = 0
	}
	from := start - 48
	if from < 0 {
		from = 0
	}
	window := text[from:start]
	m := prefixLawPattern.FindStringSubmatch(window)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func findLawFromPhrase(text string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	from := start - 80
	if from < 0 {
		from = 0
	}
	to := end + 80
	if to > len(text) {
		to = len(text)
	}
	window := strings.ToLower(text[from:to])
	for phrase, alias := range lawPhraseAliases {
		if strings.Contains(window, phrase) {
			return alias
		}
	}
	return ""
}

func normalizeLawToken(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	low := strings.ToLower(v)
	if alias, ok := lawPhraseAliases[low]; ok {
		return alias
	}
	v = strings.Trim(v, "()[]{}.,;:")
	v = strings.ToUpper(v)
	if alias, ok := lawPhraseAliases[strings.ToLower(v)]; ok {
		return alias
	}
	return v
}
