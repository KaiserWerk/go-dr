package exporter

import (
	"encoding/json"
	"io"

	"github.com/KaiserWerk/go-dr"
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
