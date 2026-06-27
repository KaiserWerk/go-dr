package bundesrecht

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/KaiserWerk/go-dr"
)

func TestParseTOC(t *testing.T) {
	raw := []byte(`<?xml version="1.0"?><items><item><title>ZPO</title><link>http://www.gesetze-im-internet.de/zpo/xml.zip</link></item></items>`)
	items, err := parseTOC(raw)
	if err != nil {
		t.Fatalf("parse toc: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Title != "ZPO" {
		t.Fatalf("unexpected title: %q", items[0].Title)
	}
}

func TestFirstXMLFromZIP(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	f, err := zw.Create("law.xml")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	_, _ = f.Write([]byte("<root/>"))
	_ = zw.Close()

	raw, err := firstXMLFromZIP(buf.Bytes())
	if err != nil {
		t.Fatalf("unzip xml: %v", err)
	}
	if strings.TrimSpace(string(raw)) != "<root/>" {
		t.Fatalf("unexpected xml content: %q", string(raw))
	}
}

func TestSourceListDocuments(t *testing.T) {
	toc := `<?xml version="1.0"?><items><item><title>ZPO</title><link>http://www.gesetze-im-internet.de/zpo/xml.zip</link></item></items>`
	s := Source{Client: newTestClient([]byte(toc))}
	refs, err := s.ListDocuments(context.Background())
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("unexpected refs count: %d", len(refs))
	}
	if refs[0].ID != "zpo" {
		t.Fatalf("unexpected ref id: %q", refs[0].ID)
	}
	if refs[0].Metadata["title"] != "ZPO" {
		t.Fatalf("unexpected ref title: %q", refs[0].Metadata["title"])
	}
}

func TestSourceFetchDocument(t *testing.T) {
	xmlBody := `<dokumente><norm doknr="A"><metadaten><jurabk>ZPO</jurabk><enbez>§ 1</enbez><titel>Sachliche Zuständigkeit</titel></metadaten><textdaten><text><Content><P>Text A.</P></Content></text></textdaten></norm></dokumente>`
	zipBody := makeZIP(t, "law.xml", []byte(xmlBody))

	s := Source{Client: newTestClient(zipBody)}
	doc, err := s.FetchDocument(context.Background(), godr.DocumentRef{ID: "zpo", URL: "https://www.gesetze-im-internet.de/zpo/xml.zip"})
	if err != nil {
		t.Fatalf("fetch document: %v", err)
	}
	if doc.ID != "zpo" {
		t.Fatalf("unexpected doc id: %q", doc.ID)
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("unexpected section count: %d", len(doc.Sections))
	}
}

func newTestClient(body []byte) *crawlerClientAdapter {
	return &crawlerClientAdapter{body: body}
}

type crawlerClientAdapter struct {
	body []byte
}

func (a *crawlerClientAdapter) Get(context.Context, string) (int, []byte, error) {
	return http.StatusOK, a.body, nil
}

func makeZIP(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	f, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
