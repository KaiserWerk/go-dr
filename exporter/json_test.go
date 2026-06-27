package exporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/KaiserWerk/go-dr"
)

func TestMarshalDocument(t *testing.T) {
	raw, err := MarshalDocument(&godr.LegalDocument{ID: "x", Title: "Doc"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"ID": "x"`) {
		t.Fatalf("unexpected payload: %s", string(raw))
	}
}

func TestWriteDocument(t *testing.T) {
	buf := new(bytes.Buffer)
	err := WriteDocument(buf, &godr.LegalDocument{ID: "x"})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), `"ID": "x"`) {
		t.Fatalf("unexpected json output: %s", buf.String())
	}
}
