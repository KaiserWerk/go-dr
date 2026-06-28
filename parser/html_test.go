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

func TestHTMLDocumentParser_ParseMetaDates(t *testing.T) {
	raw := []byte(`<!doctype html><html><head><title>X</title><meta name="datePublished" content="2020-01-02"/><meta name="effectiveFrom" content="03.04.2021"/><meta name="effectiveTo" content="20230405"/></head><body><section id="s1"><h2>A</h2><p>B</p></section></body></html>`)

	p := HTMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse html dates: %v", err)
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
}

func TestHTMLDocumentParser_ParseBodyDates(t *testing.T) {
	raw := []byte(`<!doctype html><html><head><title>X</title></head><body><p>Verkuendet: 01.02.2000. Inkrafttreten: 2000-03-04. Ausserkrafttreten: 20991231.</p><section id="s1"><h2>A</h2><p>B</p></section></body></html>`)

	p := HTMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse html body dates: %v", err)
	}

	if doc.PublishedAt == nil || doc.PublishedAt.Format("2006-01-02") != "2000-02-01" {
		t.Fatalf("unexpected PublishedAt: %v", doc.PublishedAt)
	}
	if doc.EffectiveFrom == nil || doc.EffectiveFrom.Format("2006-01-02") != "2000-03-04" {
		t.Fatalf("unexpected EffectiveFrom: %v", doc.EffectiveFrom)
	}
	if doc.EffectiveTo == nil || doc.EffectiveTo.Format("2006-01-02") != "2099-12-31" {
		t.Fatalf("unexpected EffectiveTo: %v", doc.EffectiveTo)
	}
}

func TestHTMLDocumentParser_MetaBodyConflictPrefersValidRange(t *testing.T) {
	raw := []byte(`<!doctype html><html><head><title>X</title><meta name="effectiveFrom" content="2022-01-01"/><meta name="effectiveTo" content="2021-01-01"/></head><body><p>Ausserkrafttreten: 2023-12-31.</p><section id="s1"><h2>A</h2><p>B</p></section></body></html>`)

	p := HTMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse html conflict dates: %v", err)
	}

	if doc.EffectiveFrom == nil || doc.EffectiveFrom.Format("2006-01-02") != "2022-01-01" {
		t.Fatalf("unexpected EffectiveFrom: %v", doc.EffectiveFrom)
	}
	if doc.EffectiveTo == nil || doc.EffectiveTo.Format("2006-01-02") != "2023-12-31" {
		t.Fatalf("unexpected EffectiveTo: %v", doc.EffectiveTo)
	}
}

func TestHTMLDocumentParser_InvalidRangeDropsEffectiveTo(t *testing.T) {
	raw := []byte(`<!doctype html><html><head><title>X</title><meta name="effectiveFrom" content="2024-01-01"/><meta name="effectiveTo" content="2023-01-01"/></head><body><section id="s1"><h2>A</h2><p>B</p></section></body></html>`)

	p := HTMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse html invalid range: %v", err)
	}

	if doc.EffectiveFrom == nil || doc.EffectiveFrom.Format("2006-01-02") != "2024-01-01" {
		t.Fatalf("unexpected EffectiveFrom: %v", doc.EffectiveFrom)
	}
	if doc.EffectiveTo != nil {
		t.Fatalf("expected EffectiveTo=nil for invalid range, got: %v", doc.EffectiveTo)
	}
}
