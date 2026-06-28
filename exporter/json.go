package exporter

import (
	"bytes"
	"encoding/json"
	"io"

	godr "github.com/KaiserWerk/go-dr"
)

// MarshalDocument encodes one legal document to pretty JSON.
func MarshalDocument(doc *godr.LegalDocument) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

// WriteDocument writes one legal document as JSON to w.
func WriteDocument(w io.Writer, doc *godr.LegalDocument) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// MarshalDocuments encodes many legal documents to pretty JSON.
func MarshalDocuments(docs []*godr.LegalDocument) ([]byte, error) {
	return json.MarshalIndent(docs, "", "  ")
}

// MarshalDocumentsJSONL encodes many legal documents in JSONL format.
func MarshalDocumentsJSONL(docs []*godr.LegalDocument) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := WriteDocumentsJSONL(buf, docs); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteDocumentsJSONL writes many legal documents as one compact JSON object per line.
func WriteDocumentsJSONL(w io.Writer, docs []*godr.LegalDocument) error {
	enc := json.NewEncoder(w)
	for _, doc := range docs {
		if err := enc.Encode(doc); err != nil {
			return err
		}
	}
	return nil
}
