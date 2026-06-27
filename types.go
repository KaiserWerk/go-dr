package godr

import "time"

// DocumentType describes the legal document category.
type DocumentType string

const (
	DocumentTypeUnknown DocumentType = "unknown"
	DocumentTypeLaw     DocumentType = "law"
	DocumentTypeOrder   DocumentType = "order"
	DocumentTypeRuling  DocumentType = "ruling"
)

// ReferenceType describes a relation target type found in legal text.
type ReferenceType string

const (
	ReferenceTypeUnknown   ReferenceType = "unknown"
	ReferenceTypeParagraph ReferenceType = "paragraph"
	ReferenceTypeArticle   ReferenceType = "article"
	ReferenceTypeLaw       ReferenceType = "law"
)

// DocumentRef is the minimum source-specific identifier required to fetch a document.
type DocumentRef struct {
	ID        string
	URL       string
	SourceKey string
	Metadata  map[string]string
}

// Reference points to another legal resource.
type Reference struct {
	Type   ReferenceType
	Target string
}

// Section is a normalized unit of legal content.
type Section struct {
	Identifier string
	Heading    string
	Content    string
	References []Reference
}

// LegalDocument is the normalized representation consumed by downstream systems.
type LegalDocument struct {
	ID            string
	Title         string
	ShortTitle    string
	Jurisdiction  string
	Type          DocumentType
	PublishedAt   *time.Time
	EffectiveFrom *time.Time
	EffectiveTo   *time.Time
	SourceURL     string
	Sections      []Section
	Metadata      map[string]string
}

// Clone returns a deep copy that can be modified safely by callers.
func (d *LegalDocument) Clone() *LegalDocument {
	if d == nil {
		return nil
	}

	clone := *d
	if d.Metadata != nil {
		clone.Metadata = make(map[string]string, len(d.Metadata))
		for k, v := range d.Metadata {
			clone.Metadata[k] = v
		}
	}

	if d.Sections != nil {
		clone.Sections = make([]Section, len(d.Sections))
		for i := range d.Sections {
			clone.Sections[i] = d.Sections[i]
			if d.Sections[i].References != nil {
				clone.Sections[i].References = make([]Reference, len(d.Sections[i].References))
				copy(clone.Sections[i].References, d.Sections[i].References)
			}
		}
	}

	return &clone
}
