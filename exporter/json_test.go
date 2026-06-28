package exporter

import (
	"bytes"
	"strings"
	"testing"

	godr "github.com/KaiserWerk/go-dr"
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

func TestMarshalDocumentsJSONL(t *testing.T) {
	raw, err := MarshalDocumentsJSONL([]*godr.LegalDocument{{ID: "a"}, {ID: "b"}})
	if err != nil {
		t.Fatalf("marshal jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected line count: %d", len(lines))
	}
	if !strings.Contains(lines[0], `"ID":"a"`) {
		t.Fatalf("unexpected first line: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"ID":"b"`) {
		t.Fatalf("unexpected second line: %s", lines[1])
	}
}

func TestWriteDocumentsJSONL(t *testing.T) {
	buf := new(bytes.Buffer)
	err := WriteDocumentsJSONL(buf, []*godr.LegalDocument{{ID: "x"}})
	if err != nil {
		t.Fatalf("write jsonl: %v", err)
	}
	if !strings.Contains(strings.TrimSpace(buf.String()), `"ID":"x"`) {
		t.Fatalf("unexpected jsonl output: %s", buf.String())
	}
}
