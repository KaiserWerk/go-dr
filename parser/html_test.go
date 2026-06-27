package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHTMLDocumentParser_Parse(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "sample.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	p := HTMLDocumentParser{}
	doc, err := p.Parse(data)
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}

	if doc.Title != "Buergerliches Gesetzbuch" {
		t.Fatalf("unexpected title: %q", doc.Title)
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("unexpected sections count: %d", len(doc.Sections))
	}
	if doc.Sections[0].Identifier != "535" {
		t.Fatalf("unexpected section id: %q", doc.Sections[0].Identifier)
	}
}
