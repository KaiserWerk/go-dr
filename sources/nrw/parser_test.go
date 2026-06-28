package nrw

import "testing"

func TestHTMLDocumentParser_Parse(t *testing.T) {
	raw := []byte(`<!doctype html><html><head><title>Bauordnung (BauO NRW)</title></head><body><h1>Bauordnung (BauO NRW)</h1><section id="p1"><h2>Allgemeines</h2><p>Siehe § 1 BauO NRW i. V. m. Art. 3 GG.</p></section></body></html>`)

	p := HTMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	if doc.ShortTitle != "BauO NRW" {
		t.Fatalf("unexpected short title: %q", doc.ShortTitle)
	}
	if doc.Jurisdiction != "DE-NW" {
		t.Fatalf("unexpected jurisdiction: %q", doc.Jurisdiction)
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("unexpected sections count: %d", len(doc.Sections))
	}
	if len(doc.Sections[0].References) < 2 {
		t.Fatalf("unexpected references count: %d", len(doc.Sections[0].References))
	}
}
