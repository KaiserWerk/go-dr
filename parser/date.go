package parser

import (
	"regexp"
	"strings"
	"time"
)

var anyDatePattern = regexp.MustCompile(`\b(?:\d{4}-\d{2}-\d{2}|\d{1,2}\.\d{1,2}\.\d{4}|\d{8})\b`)

func parseDateLoose(v string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}

	for _, layout := range []string{time.RFC3339, "2006-01-02", "02.01.2006", "2.1.2006", "20060102"} {
		if ts, err := time.Parse(layout, v); err == nil {
			t := ts.UTC()
			return &t
		}
	}

	match := anyDatePattern.FindString(v)
	if match == "" || match == v {
		return nil
	}
	return parseDateLoose(match)
}
