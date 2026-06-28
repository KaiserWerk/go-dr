package exporter

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	godr "github.com/KaiserWerk/go-dr"
)

const defaultPostgresSchema = "godr"

// PostgresStore persists normalized legal documents in PostgreSQL.
type PostgresStore struct {
	DB     *sql.DB
	Schema string
}

// EmbeddingRecord is one persisted embedding row for pgvector-backed retrieval.
type EmbeddingRecord struct {
	DocumentID   string
	SectionIndex int
	ChunkIndex   int
	Content      string
	Embedding    []float32
	Metadata     map[string]string
}

// SimilarChunk is one similarity-search hit returned from pgvector queries.
type SimilarChunk struct {
	DocumentID   string
	SectionIndex int
	ChunkIndex   int
	Content      string
	Metadata     map[string]string
	Distance     float64
}

// NewPostgresStore creates a persistence store using the default schema.
func NewPostgresStore(db *sql.DB) PostgresStore {
	return PostgresStore{DB: db, Schema: defaultPostgresSchema}
}

// WithSchema returns a copy configured for a different schema.
func (s PostgresStore) WithSchema(schema string) PostgresStore {
	s.Schema = strings.TrimSpace(schema)
	if s.Schema == "" {
		s.Schema = defaultPostgresSchema
	}
	return s
}

// InitSchema creates the base persistence schema and tables.
func (s PostgresStore) InitSchema(ctx context.Context) error {
	if err := s.validate(); err != nil {
		return err
	}

	docsTable := s.qualifiedTable("documents")
	sectionsTable := s.qualifiedTable("sections")
	versionsTable := s.qualifiedTable("versions")

	statements := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, s.quotedSchema()),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id TEXT PRIMARY KEY,
				source_key TEXT NOT NULL DEFAULT '',
				title TEXT NOT NULL DEFAULT '',
				short_title TEXT NOT NULL DEFAULT '',
				jurisdiction TEXT NOT NULL DEFAULT '',
				type TEXT NOT NULL DEFAULT '',
				published_at TIMESTAMPTZ NULL,
				effective_from TIMESTAMPTZ NULL,
				effective_to TIMESTAMPTZ NULL,
				source_url TEXT NOT NULL DEFAULT '',
				metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
				document_json JSONB NOT NULL DEFAULT '{}'::jsonb,
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)
		`, docsTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				document_id TEXT NOT NULL REFERENCES %s(id) ON DELETE CASCADE,
				section_index INTEGER NOT NULL,
				identifier TEXT NOT NULL DEFAULT '',
				heading TEXT NOT NULL DEFAULT '',
				content TEXT NOT NULL DEFAULT '',
				references JSONB NOT NULL DEFAULT '[]'::jsonb,
				norm_chains JSONB NOT NULL DEFAULT '[]'::jsonb,
				PRIMARY KEY (document_id, section_index)
			)
		`, sectionsTable, docsTable),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				document_id TEXT NOT NULL REFERENCES %s(id) ON DELETE CASCADE,
				version_index INTEGER NOT NULL,
				valid_from TIMESTAMPTZ NULL,
				valid_to TIMESTAMPTZ NULL,
				PRIMARY KEY (document_id, version_index)
			)
		`, versionsTable, docsTable),
	}

	for _, stmt := range statements {
		if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

// InitVectorSchema creates pgvector extension and embedding table.
func (s PostgresStore) InitVectorSchema(ctx context.Context, dimensions int) error {
	if dimensions <= 0 {
		return errors.New("dimensions must be > 0")
	}
	if err := s.InitSchema(ctx); err != nil {
		return err
	}

	embeddingsTable := s.qualifiedTable("embeddings")
	docsTable := s.qualifiedTable("documents")

	statements := []string{
		"CREATE EXTENSION IF NOT EXISTS vector",
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				document_id TEXT NOT NULL REFERENCES %s(id) ON DELETE CASCADE,
				section_index INTEGER NOT NULL,
				chunk_index INTEGER NOT NULL,
				content TEXT NOT NULL DEFAULT '',
				embedding VECTOR(%d) NOT NULL,
				metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				PRIMARY KEY (document_id, section_index, chunk_index)
			)
		`, embeddingsTable, docsTable, dimensions),
	}

	for _, stmt := range statements {
		if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

// UpsertDocument stores one document and fully replaces section/version rows.
func (s PostgresStore) UpsertDocument(ctx context.Context, sourceKey string, doc *godr.LegalDocument) error {
	if err := s.validate(); err != nil {
		return err
	}
	if doc == nil {
		return errors.New("document is nil")
	}

	docID := persistableDocumentID(doc)
	if docID == "" {
		return errors.New("document requires ID, title, or source URL")
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := upsertDocumentTx(ctx, tx, s, strings.TrimSpace(sourceKey), docID, doc.Clone()); err != nil {
		return err
	}

	return tx.Commit()
}

// UpsertDocuments stores many documents in one transaction.
func (s PostgresStore) UpsertDocuments(ctx context.Context, sourceKey string, docs []*godr.LegalDocument) error {
	if err := s.validate(); err != nil {
		return err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, doc := range docs {
		if doc == nil {
			continue
		}
		docID := persistableDocumentID(doc)
		if docID == "" {
			return errors.New("document requires ID, title, or source URL")
		}
		if err := upsertDocumentTx(ctx, tx, s, strings.TrimSpace(sourceKey), docID, doc.Clone()); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpsertEmbedding stores one vectorized chunk row in the embedding table.
func (s PostgresStore) UpsertEmbedding(ctx context.Context, row EmbeddingRecord) error {
	if err := s.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(row.DocumentID) == "" {
		return errors.New("embedding requires document ID")
	}
	vectorLiteral, err := toVectorLiteral(row.Embedding)
	if err != nil {
		return err
	}

	metadata, err := marshalJSONOrDefault(row.Metadata, map[string]string{})
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		INSERT INTO %s
			(document_id, section_index, chunk_index, content, embedding, metadata, updated_at)
		VALUES ($1, $2, $3, $4, $5::vector, $6::jsonb, NOW())
		ON CONFLICT (document_id, section_index, chunk_index)
		DO UPDATE SET
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, s.qualifiedTable("embeddings"))

	_, err = s.DB.ExecContext(
		ctx,
		query,
		strings.TrimSpace(row.DocumentID),
		row.SectionIndex,
		row.ChunkIndex,
		strings.TrimSpace(row.Content),
		vectorLiteral,
		metadata,
	)
	return err
}

// SearchSimilarChunks queries nearest embeddings ordered by cosine distance.
func (s PostgresStore) SearchSimilarChunks(ctx context.Context, queryEmbedding []float32, limit int) ([]SimilarChunk, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	vectorLiteral, err := toVectorLiteral(queryEmbedding)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	q := fmt.Sprintf(`
		SELECT document_id, section_index, chunk_index, content, metadata, embedding <=> $1::vector AS distance
		FROM %s
		ORDER BY embedding <=> $1::vector
		LIMIT $2
	`, s.qualifiedTable("embeddings"))

	rows, err := s.DB.QueryContext(ctx, q, vectorLiteral, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SimilarChunk, 0)
	for rows.Next() {
		var item SimilarChunk
		var rawMetadata []byte
		if err := rows.Scan(
			&item.DocumentID,
			&item.SectionIndex,
			&item.ChunkIndex,
			&item.Content,
			&rawMetadata,
			&item.Distance,
		); err != nil {
			return nil, err
		}
		if len(rawMetadata) > 0 {
			_ = json.Unmarshal(rawMetadata, &item.Metadata)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// EnsureVectorCosineIndex creates an ivfflat cosine index for the embeddings table.
func (s PostgresStore) EnsureVectorCosineIndex(ctx context.Context, indexName string, lists int) error {
	if err := s.validate(); err != nil {
		return err
	}
	indexName = strings.TrimSpace(indexName)
	if indexName == "" {
		indexName = "embeddings_embedding_cosine_ivfflat"
	}
	lists = max(lists, 1)

	stmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s USING ivfflat (embedding vector_cosine_ops) WITH (lists = %d)`,
		quoteIdentifier(indexName),
		s.qualifiedTable("embeddings"),
		lists,
	)
	_, err := s.DB.ExecContext(ctx, stmt)
	return err
}

func upsertDocumentTx(ctx context.Context, tx *sql.Tx, s PostgresStore, sourceKey, docID string, doc *godr.LegalDocument) error {
	if doc == nil {
		return errors.New("document is nil")
	}
	doc.ID = docID
	doc.EnsureVersions()

	metadata, err := marshalJSONOrDefault(doc.Metadata, map[string]string{})
	if err != nil {
		return err
	}
	docJSON, err := marshalJSONOrDefault(doc, &godr.LegalDocument{})
	if err != nil {
		return err
	}

	insDoc := fmt.Sprintf(`
		INSERT INTO %s
			(id, source_key, title, short_title, jurisdiction, type, published_at, effective_from, effective_to, source_url, metadata, document_json, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, NOW())
		ON CONFLICT (id)
		DO UPDATE SET
			source_key = EXCLUDED.source_key,
			title = EXCLUDED.title,
			short_title = EXCLUDED.short_title,
			jurisdiction = EXCLUDED.jurisdiction,
			type = EXCLUDED.type,
			published_at = EXCLUDED.published_at,
			effective_from = EXCLUDED.effective_from,
			effective_to = EXCLUDED.effective_to,
			source_url = EXCLUDED.source_url,
			metadata = EXCLUDED.metadata,
			document_json = EXCLUDED.document_json,
			updated_at = NOW()
	`, s.qualifiedTable("documents"))

	_, err = tx.ExecContext(
		ctx,
		insDoc,
		doc.ID,
		sourceKey,
		strings.TrimSpace(doc.Title),
		strings.TrimSpace(doc.ShortTitle),
		strings.TrimSpace(doc.Jurisdiction),
		string(doc.Type),
		nullableTime(doc.PublishedAt),
		nullableTime(doc.EffectiveFrom),
		nullableTime(doc.EffectiveTo),
		strings.TrimSpace(doc.SourceURL),
		metadata,
		docJSON,
	)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE document_id = $1`, s.qualifiedTable("sections")), doc.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE document_id = $1`, s.qualifiedTable("versions")), doc.ID); err != nil {
		return err
	}

	insSection := fmt.Sprintf(`
		INSERT INTO %s
			(document_id, section_index, identifier, heading, content, references, norm_chains)
		VALUES
			($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb)
	`, s.qualifiedTable("sections"))

	for i, sec := range doc.Sections {
		refsJSON, err := marshalJSONOrDefault(sec.References, []godr.Reference{})
		if err != nil {
			return err
		}
		chainsJSON, err := marshalJSONOrDefault(sec.NormChains, []godr.NormChain{})
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			insSection,
			doc.ID,
			i,
			strings.TrimSpace(sec.Identifier),
			strings.TrimSpace(sec.Heading),
			strings.TrimSpace(sec.Content),
			refsJSON,
			chainsJSON,
		); err != nil {
			return err
		}
	}

	insVersion := fmt.Sprintf(`
		INSERT INTO %s
			(document_id, version_index, valid_from, valid_to)
		VALUES
			($1, $2, $3, $4)
	`, s.qualifiedTable("versions"))

	for i, v := range doc.Versions {
		var validFrom any
		if !v.ValidFrom.IsZero() {
			validFrom = v.ValidFrom.UTC()
		}
		if _, err := tx.ExecContext(ctx, insVersion, doc.ID, i, validFrom, nullableTime(v.ValidTo)); err != nil {
			return err
		}
	}

	return nil
}

func persistableDocumentID(doc *godr.LegalDocument) string {
	if doc == nil {
		return ""
	}
	if id := strings.TrimSpace(doc.ID); id != "" {
		return id
	}

	basis := strings.TrimSpace(doc.SourceURL)
	if basis == "" {
		basis = strings.TrimSpace(doc.Title)
	}
	if basis == "" {
		return ""
	}

	sum := sha1.Sum([]byte(basis))
	return "doc-" + hex.EncodeToString(sum[:8])
}

func toVectorLiteral(v []float32) (string, error) {
	if len(v) == 0 {
		return "", errors.New("embedding must not be empty")
	}
	b := strings.Builder{}
	b.WriteString("[")
	for i, n := range v {
		f := float64(n)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return "", errors.New("embedding contains non-finite values")
		}
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(strconv.FormatFloat(f, 'f', -1, 32))
	}
	b.WriteString("]")
	return b.String(), nil
}

func marshalJSONOrDefault(v any, fallback any) ([]byte, error) {
	if v == nil {
		return json.Marshal(fallback)
	}
	return json.Marshal(v)
}

func nullableTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC()
}

func (s PostgresStore) validate() error {
	if s.DB == nil {
		return errors.New("postgres store requires DB")
	}
	if strings.TrimSpace(s.Schema) == "" {
		s.Schema = defaultPostgresSchema
	}
	return nil
}

func (s PostgresStore) quotedSchema() string {
	return quoteIdentifier(s.schema())
}

func (s PostgresStore) qualifiedTable(table string) string {
	return fmt.Sprintf("%s.%s", quoteIdentifier(s.schema()), quoteIdentifier(table))
}

func (s PostgresStore) schema() string {
	if strings.TrimSpace(s.Schema) == "" {
		return defaultPostgresSchema
	}
	return strings.TrimSpace(s.Schema)
}

func quoteIdentifier(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return `""`
	}
	v = strings.ReplaceAll(v, `"`, `""`)
	return `"` + v + `"`
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
