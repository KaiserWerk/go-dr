package bundesrecht

import "testing"

func TestXMLDocumentParser_Parse(t *testing.T) {
	raw := []byte(`<dokumente><norm doknr="A"><metadaten><jurabk>ZPO</jurabk><enbez>§ 1</enbez><titel>Sachliche Zuständigkeit</titel></metadaten><textdaten><text><Content><P>Text A.</P><P>Siehe § 2 BGB und Art. 3 GG.</P></Content></text></textdaten></norm></dokumente>`)

	p := XMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse xml: %v", err)
	}

	if doc.ShortTitle != "ZPO" {
		t.Fatalf("unexpected short title: %q", doc.ShortTitle)
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("unexpected sections count: %d", len(doc.Sections))
	}
	if doc.Sections[0].Identifier != "§ 1" {
		t.Fatalf("unexpected identifier: %q", doc.Sections[0].Identifier)
	}
	if len(doc.Sections[0].References) != 4 {
		t.Fatalf("unexpected reference count: %d", len(doc.Sections[0].References))
	}
	if doc.Sections[0].References[0].Target != "BGB" {
		t.Fatalf("unexpected first reference: %q", doc.Sections[0].References[0].Target)
	}
	if doc.Sections[0].References[1].Target != "BGB § 2" {
		t.Fatalf("unexpected second reference: %q", doc.Sections[0].References[1].Target)
	}
	if doc.Sections[0].References[2].Target != "GG" {
		t.Fatalf("unexpected third reference: %q", doc.Sections[0].References[2].Target)
	}
	if doc.Sections[0].References[3].Target != "GG Art. 3" {
		t.Fatalf("unexpected fourth reference: %q", doc.Sections[0].References[3].Target)
	}
}

func TestXMLDocumentParser_ParseChainAndMultiParagraph(t *testing.T) {
	raw := []byte(`<dokumente><norm doknr="A"><metadaten><jurabk>BGB</jurabk><enbez>§ 1</enbez><titel>Kette</titel></metadaten><textdaten><text><Content><P>Nach §§ 535, 536 BGB i. V. m. § 280 Abs. 1 gilt dies.</P></Content></text></textdaten></norm></dokumente>`)

	p := XMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse xml: %v", err)
	}

	if len(doc.Sections) != 1 {
		t.Fatalf("unexpected sections count: %d", len(doc.Sections))
	}

	refs := doc.Sections[0].References
	if len(refs) < 4 {
		t.Fatalf("expected at least 4 references, got %d", len(refs))
	}

	has := map[string]bool{}
	for _, r := range refs {
		has[string(r.Type)+":"+r.Target] = true
	}

	if !has["law:BGB"] {
		t.Fatalf("missing law reference BGB")
	}
	if !has["paragraph:BGB § 535"] {
		t.Fatalf("missing paragraph reference BGB § 535")
	}
	if !has["paragraph:BGB § 536"] {
		t.Fatalf("missing paragraph reference BGB § 536")
	}
	if !has["paragraph:BGB § 280 Abs. 1"] {
		t.Fatalf("missing propagated chain reference BGB § 280 Abs. 1")
	}
}

func TestXMLDocumentParser_ParseMetadataDates(t *testing.T) {
	raw := []byte(`<dokumente><norm doknr="A"><metadaten><jurabk>BGB</jurabk><enbez>§ 1</enbez><titel>Gueltigkeit</titel><ausfertigung-datum>1950-01-02</ausfertigung-datum><inkrafttreten-datum>03.04.1951</inkrafttreten-datum><ausserkrafttreten>20991231</ausserkrafttreten></metadaten><textdaten><text><Content><P>Text.</P></Content></text></textdaten></norm></dokumente>`)

	p := XMLDocumentParser{}
	doc, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("parse xml with metadata dates: %v", err)
	}

	if doc.PublishedAt == nil || doc.PublishedAt.Format("2006-01-02") != "1950-01-02" {
		t.Fatalf("unexpected PublishedAt: %v", doc.PublishedAt)
	}
	if doc.EffectiveFrom == nil || doc.EffectiveFrom.Format("2006-01-02") != "1951-04-03" {
		t.Fatalf("unexpected EffectiveFrom: %v", doc.EffectiveFrom)
	}
	if doc.EffectiveTo == nil || doc.EffectiveTo.Format("2006-01-02") != "2099-12-31" {
		t.Fatalf("unexpected EffectiveTo: %v", doc.EffectiveTo)
	}
}
