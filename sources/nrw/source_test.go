package nrw

import (
	"context"
	"errors"
	"net/http"
	"testing"

	godr "github.com/KaiserWerk/go-dr"
)

type fakeClient struct {
	body       []byte
	statusCode int
	err        error
}

func (f fakeClient) Get(context.Context, string) (int, []byte, error) {
	if f.err != nil {
		return 0, nil, f.err
	}
	if f.statusCode == 0 {
		f.statusCode = http.StatusOK
	}
	return f.statusCode, f.body, nil
}

func TestSourceListDocuments(t *testing.T) {
	html := []byte(`<!doctype html><html><body>
	<a href="/lmi/owa/br_text_anzeigen?v_id=100000000000">BauO NRW</a>
	<a href="/lmi/owa/br_text_anzeigen?v_id=100000000000">BauO NRW dup</a>
	<a href="https://example.org/foo">Ignore</a>
	</body></html>`)

	s := Source{
		Client:  fakeClient{body: html},
		ListURL: "https://recht.nrw.de/start",
	}

	refs, err := s.ListDocuments(context.Background())
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("unexpected refs count: %d", len(refs))
	}
	if refs[0].URL != "https://recht.nrw.de/lmi/owa/br_text_anzeigen?v_id=100000000000" {
		t.Fatalf("unexpected ref url: %q", refs[0].URL)
	}
}

func TestSourceFetchDocument(t *testing.T) {
	html := []byte(`<!doctype html><html><body><h1>BauO NRW</h1><section><h2>Abschnitt</h2><p>Text.</p></section></body></html>`)
	s := Source{Client: fakeClient{body: html}}

	doc, err := s.FetchDocument(context.Background(), godr.DocumentRef{ID: "bauo", URL: "https://recht.nrw.de/lmi/owa/br_text_anzeigen?v_id=123"})
	if err != nil {
		t.Fatalf("fetch document: %v", err)
	}
	if doc.ID != "bauo" {
		t.Fatalf("unexpected doc id: %q", doc.ID)
	}
	if doc.Jurisdiction != "DE-NW" {
		t.Fatalf("unexpected jurisdiction: %q", doc.Jurisdiction)
	}
}

func TestSourceFetchDocumentRequiresURL(t *testing.T) {
	s := Source{}
	_, err := s.FetchDocument(context.Background(), godr.DocumentRef{ID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSourceListDocumentsPropagatesClientError(t *testing.T) {
	s := Source{Client: fakeClient{err: errors.New("boom")}}
	_, err := s.ListDocuments(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
