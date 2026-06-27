package parser

import "github.com/KaiserWerk/go-dr"

// Parser parses raw payloads from a source into a normalized legal document.
type Parser interface {
	Parse(raw []byte) (*godr.LegalDocument, error)
}
