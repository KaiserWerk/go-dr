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
- Phase 8 (start): direct reference extraction during section parsing
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
	- uses the same normalized reference extraction pipeline as Bundesrecht
- `github.com/KaiserWerk/go-dr/sources/juris`:
	- shared adapter for Juris-based state portals
	- configurable base URL, selector, jurisdiction, and listing URL
	- preset constructors for BW, BE, HE, MV, RP, ST, SH, TH
- `github.com/KaiserWerk/go-dr/exporter`:
	- JSON marshaling helpers for single and multiple documents
	- JSONL helpers for one-document-per-line exports

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

## Tests

```bash
go test ./...
```
