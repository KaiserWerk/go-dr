package bundesrecht

import (
	"context"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/exporter"
)

// PostgresSink writes fetched Bundesrecht documents into exporter.PostgresStore.
type PostgresSink struct {
	Store     exporter.PostgresStore
	SourceKey string
}

// NewPostgresSink creates a ready-to-use sink adapter for PostgreSQL persistence.
func NewPostgresSink(store exporter.PostgresStore) PostgresSink {
	return PostgresSink{Store: store}
}

// SaveDocument implements DocumentSink.
func (s PostgresSink) SaveDocument(ctx context.Context, sourceName string, ref godr.DocumentRef, doc *godr.LegalDocument) error {
	sourceKey := strings.TrimSpace(s.SourceKey)
	if sourceKey == "" {
		sourceKey = strings.TrimSpace(ref.SourceKey)
	}
	if sourceKey == "" {
		sourceKey = strings.TrimSpace(sourceName)
	}
	return s.Store.UpsertDocument(ctx, sourceKey, doc)
}

// PostgresPipelineConfig combines regular pipeline settings with PostgreSQL persistence.
type PostgresPipelineConfig struct {
	PipelineConfig
	Store     exporter.PostgresStore
	SourceKey string
}

// RunPipelineToPostgres runs Bundesrecht completeness ingestion and persists each document in PostgreSQL.
func RunPipelineToPostgres(ctx context.Context, cfg PostgresPipelineConfig) (CompletenessReport, error) {
	base := cfg.PipelineConfig
	base.Sink = PostgresSink{Store: cfg.Store, SourceKey: cfg.SourceKey}
	return RunPipeline(ctx, base)
}
