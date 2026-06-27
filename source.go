package godr

import "context"

// Source describes a legal portal connector.
type Source interface {
	Name() string
	ListDocuments(ctx context.Context) ([]DocumentRef, error)
	FetchDocument(ctx context.Context, ref DocumentRef) (*LegalDocument, error)
}
