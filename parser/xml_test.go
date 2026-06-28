package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	godr "github.com/KaiserWerk/go-dr"
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

func TestXMLDocumentParser_ParseDates(t *testing.T) {
	raw := []byte(`<law id="x"><title>X</title><shortTitle>XT</shortTitle><jurisdiction>DE-BUND</jurisdiction><type>law</type><publishedAt>2020-01-02</publishedAt><effectiveFrom>03.04.2021</effectiveFrom><effectiveTo>20230405</effectiveTo><sections><section id="1"><heading>H</heading><content>C</content></section></sections></law>`)

	p := XMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse xml with dates: %v", err)
	}

	if doc.PublishedAt == nil || doc.PublishedAt.Format("2006-01-02") != "2020-01-02" {
		t.Fatalf("unexpected PublishedAt: %v", doc.PublishedAt)
	}
	if doc.EffectiveFrom == nil || doc.EffectiveFrom.Format("2006-01-02") != "2021-04-03" {
		t.Fatalf("unexpected EffectiveFrom: %v", doc.EffectiveFrom)
	}
	if doc.EffectiveTo == nil || doc.EffectiveTo.Format("2006-01-02") != "2023-04-05" {
		t.Fatalf("unexpected EffectiveTo: %v", doc.EffectiveTo)
	}

	if doc.PublishedAt.Location() != time.UTC {
		t.Fatalf("expected UTC timezone, got: %v", doc.PublishedAt.Location())
	}
}

func TestXMLDocumentParser_InvalidEffectiveRangeDropsTo(t *testing.T) {
	raw := []byte(`<law id="x"><title>X</title><shortTitle>XT</shortTitle><jurisdiction>DE-BUND</jurisdiction><type>law</type><effectiveFrom>2024-01-01</effectiveFrom><effectiveTo>2023-12-31</effectiveTo><sections><section id="1"><heading>H</heading><content>C</content></section></sections></law>`)

	p := XMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse xml invalid range: %v", err)
	}

	if doc.EffectiveFrom == nil || doc.EffectiveFrom.Format("2006-01-02") != "2024-01-01" {
		t.Fatalf("unexpected EffectiveFrom: %v", doc.EffectiveFrom)
	}
	if doc.EffectiveTo != nil {
		t.Fatalf("expected EffectiveTo=nil for invalid range, got: %v", doc.EffectiveTo)
	}
}
