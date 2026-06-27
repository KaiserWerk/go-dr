package bundesrecht

import "testing"

func TestXMLDocumentParser_Parse(t *testing.T) {
	raw := []byte(`<dokumente><norm doknr="A"><metadaten><jurabk>ZPO</jurabk><enbez>§ 1</enbez><titel>Sachliche Zuständigkeit</titel></metadaten><textdaten><text><Content><P>Text A.</P><P>Text B.</P></Content></text></textdaten></norm></dokumente>`)

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
}
