# go-dr

Reusable Go library for crawling and normalizing German legal documents.

## Status

Implemented foundations from the plan:

- Phase 1: source interface and modular package boundaries
- Phase 2: normalized legal data model
- Phase 3: crawler client abstractions (retry, rate limit, snapshot)
- Phase 4: parser interface plus HTML/XML base parsers and offline tests
- Phase 5 (start): Bundesrecht source with TOC listing and XML ZIP fetch/parse
- Phase 6 (start): Juris-based state source adapter skeleton
- Phase 7 (start): NRW source with dedicated list/result-page extraction
- Phase 7 (start): BAYERN.RECHT, BRAVORS, REVOSax, and VORIS source adapters
- Phase 8 (start): section-level reference extraction plus directed norm chains
- Phase 9 (start): version validity model and stichtag lookup helper
- Phase 9 (start): XML metadata date extraction for effective version windows
- Phase 10 (start): JSON and JSONL export package

No executable binaries are included.

## Installation

```bash
go get github.com/KaiserWerk/go-dr
```

## Package Overview

- `github.com/KaiserWerk/go-dr`:
	- core types (`LegalDocument`, `Section`, `Reference`)
	- enums (`DocumentType`, `ReferenceType`)
	- interfaces (`Source`)
- `github.com/KaiserWerk/go-dr/crawler`:
	- composable HTTP fetch client
	- retry policy
	- fixed-interval limiter
	- file snapshot store
- `github.com/KaiserWerk/go-dr/parser`:
	- parser interface
	- XML parser
	- HTML parser
- `github.com/KaiserWerk/go-dr/sources/bundesrecht`:
	- TOC listing from `gii-toc.xml`
	- law document download (`xml.zip`) and parsing
	- section-level extraction of paragraph/article/law references
	- support for chained references (for example i. V. m.) and multi-paragraph forms (`§§ ...`)
- `github.com/KaiserWerk/go-dr/sources/nrw`:
	- dedicated parsing for NRW list/result pages
	- stronger URL/title/id extraction from NRW anchors and result rows
	- document fetch and parse pipeline for NRW legal pages
	- uses the same normalized reference extraction and chain pipeline as Bundesrecht
- `github.com/KaiserWerk/go-dr/sources/bayern`:
	- dedicated parsing for BAYERN.RECHT list/document pages
	- URL/title/id extraction and normalized references/chains
- `github.com/KaiserWerk/go-dr/sources/brandenburg`:
	- dedicated parsing for BRAVORS list/document pages
	- URL/title/id extraction and normalized references/chains
- `github.com/KaiserWerk/go-dr/sources/sachsen`:
	- dedicated parsing for REVOSax list/document pages
	- URL/title/id extraction and normalized references/chains
- `github.com/KaiserWerk/go-dr/sources/niedersachsen`:
	- dedicated parsing for VORIS list/document pages
	- URL/title/id extraction and normalized references/chains
- `github.com/KaiserWerk/go-dr/sources/juris`:
	- shared adapter for Juris-based state portals
	- configurable base URL, selector, allowed hosts, jurisdiction, listing path/query, and listing URL
	- host-aware list extraction with stronger URL/title/id normalization
	- tuned state profiles and preset constructors for BW, BE, HE, MV, RP, ST, SH, TH
	- query bootstrap helpers via profile defaults and WithListQuery(...)
	- convenience constructors like BerlinWithQuery(...), HessenWithQuery(...)
	- profile helpers: ProfileNames(), NewProfileSource(...), NewProfileSourceWithQuery(...)
- `github.com/KaiserWerk/go-dr/exporter`:
	- JSON marshaling helpers for single and multiple documents
	- JSONL helpers for one-document-per-line exports
	- PostgreSQL persistence store with schema bootstrap and transactional document upserts
	- pgvector-ready embedding persistence and similarity search helpers

## Quick Example

```go
package yourpkg

import (
		"context"
		"time"

		"github.com/KaiserWerk/go-dr/crawler"
		"github.com/KaiserWerk/go-dr/parser"
)

func fetchAndParse(url string) error {
		c := crawler.NewClient(crawler.Config{
				UserAgent: "go-dr/0.1",
				Limiter:   crawler.NewLimiter(200 * time.Millisecond),
				Retry: crawler.RetryPolicy{
						MaxAttempts: 3,
				},
		})

		_, payload, err := c.Get(context.Background(), url)
		if err != nil {
				return err
		}

		p := parser.XMLDocumentParser{}
		_, err = p.Parse(payload)
		return err
}
```

## PostgreSQL And pgvector Example

```go
package yourpkg

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"

	"github.com/KaiserWerk/go-dr/exporter"
)

func persist(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	store := exporter.NewPostgresStore(db)
	if err := store.InitSchema(ctx); err != nil {
		return err
	}
	if err := store.InitVectorSchema(ctx, 768); err != nil {
		return err
	}
	if err := store.EnsureVectorCosineIndex(ctx, "", 100); err != nil {
		return err
	}

	// Upsert documents via store.UpsertDocument(...) / store.UpsertDocuments(...)
	// Upsert embeddings via store.UpsertEmbedding(...)
	_, err = store.SearchSimilarChunks(ctx, make([]float32, 768), 10)
	return err
}
```

## Bundesrecht Completeness Pipeline Example

```go
package yourpkg

import (
	"context"

	godr "github.com/KaiserWerk/go-dr"
	"github.com/KaiserWerk/go-dr/sources/bundesrecht"
)

func ingestBundesrecht(ctx context.Context) error {
	progress := &bundesrecht.FileProgressStore{Path: "state/bundesrecht-progress.json"}

	report, err := bundesrecht.RunPipeline(ctx, bundesrecht.PipelineConfig{
		Source: bundesrecht.Source{},
		Sink: bundesrecht.SinkFunc(func(ctx context.Context, sourceName string, ref godr.DocumentRef, doc *godr.LegalDocument) error {
			// persist doc here (for example exporter.PostgresStore)
			return nil
		}),
		Progress:        progress,
		Workers:         8,
		ContinueOnError: true,
	})
	if err != nil {
		return err
	}

	_ = report
	return nil
}
```

## Tests

```bash
go test ./...
```
