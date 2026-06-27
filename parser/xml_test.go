package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KaiserWerk/go-dr"
)

func TestXMLDocumentParser_Parse(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "sample.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	p := XMLDocumentParser{}
	doc, err := p.Parse(data)
	if err != nil {
		t.Fatalf("parse xml: %v", err)
	}

	if doc.ID != "bgb_535" {
		t.Fatalf("unexpected ID: %q", doc.ID)
	}
	if doc.Title != "Buergerliches Gesetzbuch" {
		t.Fatalf("unexpected title: %q", doc.Title)
	}
	if doc.Type != godr.DocumentTypeLaw {
		t.Fatalf("unexpected type: %q", doc.Type)
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("unexpected sections count: %d", len(doc.Sections))
	}
}
