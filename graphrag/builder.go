package graphrag

import (
	"fmt"
	"strings"

	godr "github.com/KaiserWerk/go-dr"
)

// BuilderConfig controls projection of normalized documents into graph structures.
type BuilderConfig struct {
	IncludeConcretizesEdges bool
	IncludeCitesEdges       bool
	IncludeReferenceEdges   bool
	IncludeReplacesEdges    bool
}

// DefaultBuilderConfig enables all relation types supported by the current model.
func DefaultBuilderConfig() BuilderConfig {
	return BuilderConfig{
		IncludeConcretizesEdges: true,
		IncludeCitesEdges:       true,
		IncludeReferenceEdges:   true,
		IncludeReplacesEdges:    true,
	}
}

// Build converts many legal documents into one knowledge graph.
func Build(docs []*godr.LegalDocument, cfg BuilderConfig) *Graph {
	g := NewGraph()
	for _, doc := range docs {
		addDocument(g, doc, cfg)
	}
	return g
}

// BuildWithDefaults converts many legal documents using all supported edge types.
func BuildWithDefaults(docs []*godr.LegalDocument) *Graph {
	return Build(docs, DefaultBuilderConfig())
}

func addDocument(g *Graph, doc *godr.LegalDocument, cfg BuilderConfig) {
	if g == nil || doc == nil {
		return
	}

	docID := documentNodeID(doc)
	docLabel := firstNonEmpty(strings.TrimSpace(doc.ShortTitle), strings.TrimSpace(doc.Title), docID)

	docNodeType := NodeTypeLaw
	if doc.Type == godr.DocumentTypeRuling {
		docNodeType = NodeTypeRuling
	}

	g.AddNode(Node{
		ID:           docID,
		Type:         docNodeType,
		Label:        docLabel,
		Jurisdiction: strings.TrimSpace(doc.Jurisdiction),
		Metadata: map[string]string{
			"document_id": strings.TrimSpace(doc.ID),
			"source_url":  strings.TrimSpace(doc.SourceURL),
			"title":       strings.TrimSpace(doc.Title),
			"short_title": strings.TrimSpace(doc.ShortTitle),
		},
	})

	if cfg.IncludeReplacesEdges {
		for _, target := range replacedTargets(doc.Metadata) {
			targetNodeID := externalNodeID(target)
			g.AddNode(Node{ID: targetNodeID, Type: NodeTypeLaw, Label: target})
			g.AddEdge(Edge{
				From:   docID,
				To:     targetNodeID,
				Type:   EdgeTypeReplaces,
				Weight: 1,
			})
		}
	}

	for i, sec := range doc.Sections {
		sectionNodeID := sectionNodeID(docID, i, sec)
		sectionLabel := firstNonEmpty(strings.TrimSpace(sec.Identifier), strings.TrimSpace(sec.Heading), fmt.Sprintf("section-%d", i+1))

		g.AddNode(Node{
			ID:           sectionNodeID,
			Type:         NodeTypeParagraph,
			Label:        sectionLabel,
			Jurisdiction: strings.TrimSpace(doc.Jurisdiction),
			Metadata: map[string]string{
				"document_node_id": docID,
				"identifier":       strings.TrimSpace(sec.Identifier),
				"heading":          strings.TrimSpace(sec.Heading),
			},
		})

		if cfg.IncludeConcretizesEdges {
			g.AddEdge(Edge{
				From:   sectionNodeID,
				To:     docID,
				Type:   EdgeTypeConcretizes,
				Weight: 1,
			})
		}

		if cfg.IncludeCitesEdges {
			for _, ref := range sec.References {
				targetLabel := strings.TrimSpace(ref.Target)
				if targetLabel == "" {
					continue
				}
				targetNodeID := referenceNodeID(ref)
				targetType := nodeTypeFromReference(ref.Type)
				g.AddNode(Node{ID: targetNodeID, Type: targetType, Label: targetLabel})
				g.AddEdge(Edge{
					From:   sectionNodeID,
					To:     targetNodeID,
					Type:   EdgeTypeCites,
					Weight: 1,
					Metadata: map[string]string{
						"reference_type": string(ref.Type),
					},
				})
			}
		}

		if cfg.IncludeReferenceEdges {
			for _, chain := range sec.NormChains {
				srcLabel := strings.TrimSpace(chain.Source)
				dstLabel := strings.TrimSpace(chain.Target)
				if srcLabel == "" || dstLabel == "" || srcLabel == dstLabel {
					continue
				}
				srcID := externalNodeID(srcLabel)
				dstID := externalNodeID(dstLabel)
				g.AddNode(Node{ID: srcID, Type: NodeTypeParagraph, Label: srcLabel})
				g.AddNode(Node{ID: dstID, Type: NodeTypeParagraph, Label: dstLabel})
				g.AddEdge(Edge{From: srcID, To: dstID, Type: EdgeTypeReferences, Weight: 1})
			}
		}
	}
}

func documentNodeID(doc *godr.LegalDocument) string {
	if doc == nil {
		return "law:unknown"
	}
	id := strings.TrimSpace(doc.ID)
	if id == "" {
		id = strings.TrimSpace(doc.ShortTitle)
	}
	if id == "" {
		id = strings.TrimSpace(doc.Title)
	}
	if id == "" {
		id = strings.TrimSpace(doc.SourceURL)
	}
	id = normalizeNodeToken(id)
	if id == "" {
		id = "unknown"
	}
	if doc.Type == godr.DocumentTypeRuling {
		return "ruling:" + id
	}
	return "law:" + id
}

func sectionNodeID(documentNodeID string, idx int, sec godr.Section) string {
	key := strings.TrimSpace(sec.Identifier)
	if key == "" {
		key = strings.TrimSpace(sec.Heading)
	}
	if key == "" {
		key = fmt.Sprintf("section-%d", idx+1)
	}
	key = normalizeNodeToken(key)
	if key == "" {
		key = fmt.Sprintf("section-%d", idx+1)
	}
	return "paragraph:" + documentNodeID + ":" + key
}

func referenceNodeID(ref godr.Reference) string {
	prefix := "reference"
	switch ref.Type {
	case godr.ReferenceTypeLaw:
		prefix = "law"
	case godr.ReferenceTypeArticle:
		prefix = "paragraph"
	case godr.ReferenceTypeParagraph:
		prefix = "paragraph"
	}
	return prefix + ":" + normalizeNodeToken(ref.Target)
}

func externalNodeID(label string) string {
	return "paragraph:" + normalizeNodeToken(label)
}

func normalizeNodeToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	b := strings.Builder{}
	b.Grow(len(v))
	lastDash := false
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}

func nodeTypeFromReference(t godr.ReferenceType) NodeType {
	switch t {
	case godr.ReferenceTypeLaw:
		return NodeTypeLaw
	case godr.ReferenceTypeParagraph, godr.ReferenceTypeArticle:
		return NodeTypeParagraph
	default:
		return NodeTypeUnknown
	}
}

func replacedTargets(meta map[string]string) []string {
	if len(meta) == 0 {
		return nil
	}
	keyCandidates := []string{"replaces", "ersetzt", "abloesung", "ablösung"}
	for _, key := range keyCandidates {
		if raw := strings.TrimSpace(meta[key]); raw != "" {
			parts := splitMulti(raw)
			out := make([]string, 0, len(parts))
			seen := make(map[string]struct{}, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				norm := strings.ToLower(p)
				if _, ok := seen[norm]; ok {
					continue
				}
				seen[norm] = struct{}{}
				out = append(out, p)
			}
			return out
		}
	}
	return nil
}

func splitMulti(v string) []string {
	fields := strings.FieldsFunc(v, func(r rune) bool {
		switch r {
		case ',', ';', '|':
			return true
		default:
			return false
		}
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if s := strings.TrimSpace(f); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
