package graphrag

import (
	"sort"
	"strings"
)

// NodeType classifies one graph node.
type NodeType string

const (
	NodeTypeUnknown   NodeType = "unknown"
	NodeTypeLaw       NodeType = "law"
	NodeTypeParagraph NodeType = "paragraph"
	NodeTypeRuling    NodeType = "ruling"
)

// EdgeType classifies one directed relation.
type EdgeType string

const (
	EdgeTypeReferences  EdgeType = "references"
	EdgeTypeCites       EdgeType = "cites"
	EdgeTypeReplaces    EdgeType = "replaces"
	EdgeTypeConcretizes EdgeType = "concretizes"
)

// Node represents one legal resource in the graph.
type Node struct {
	ID           string
	Type         NodeType
	Label        string
	Jurisdiction string
	Metadata     map[string]string
}

// Edge represents one directed relation between two nodes.
type Edge struct {
	From     string
	To       string
	Type     EdgeType
	Weight   float64
	Metadata map[string]string
}

// Graph stores nodes and edges with deduplication and lookup helpers.
type Graph struct {
	nodes map[string]Node
	edges map[string]Edge
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]Node),
		edges: make(map[string]Edge),
	}
}

// AddNode inserts or merges one node by ID.
func (g *Graph) AddNode(n Node) {
	if g == nil {
		return
	}
	n.ID = strings.TrimSpace(n.ID)
	if n.ID == "" {
		return
	}
	n.Label = strings.TrimSpace(n.Label)
	n.Jurisdiction = strings.TrimSpace(n.Jurisdiction)

	if prev, ok := g.nodes[n.ID]; ok {
		g.nodes[n.ID] = mergeNode(prev, n)
		return
	}
	g.nodes[n.ID] = n
}

// AddEdge inserts or merges one edge.
func (g *Graph) AddEdge(e Edge) {
	if g == nil {
		return
	}
	e.From = strings.TrimSpace(e.From)
	e.To = strings.TrimSpace(e.To)
	if e.From == "" || e.To == "" || e.From == e.To {
		return
	}
	if e.Weight <= 0 {
		e.Weight = 1
	}
	key := edgeKey(e)
	if prev, ok := g.edges[key]; ok {
		merged := prev
		merged.Weight += e.Weight
		merged.Metadata = mergeMap(prev.Metadata, e.Metadata)
		g.edges[key] = merged
		return
	}
	g.edges[key] = e
}

// Node returns one node by ID.
func (g *Graph) Node(id string) (Node, bool) {
	if g == nil {
		return Node{}, false
	}
	n, ok := g.nodes[strings.TrimSpace(id)]
	return n, ok
}

// Nodes returns all nodes in stable order by ID.
func (g *Graph) Nodes() []Node {
	if g == nil {
		return nil
	}
	out := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// Edges returns all edges in stable order.
func (g *Graph) Edges() []Edge {
	if g == nil {
		return nil
	}
	out := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].Type < out[j].Type
	})
	return out
}

// Outgoing returns all outgoing edges for a node.
func (g *Graph) Outgoing(nodeID string) []Edge {
	if g == nil {
		return nil
	}
	nodeID = strings.TrimSpace(nodeID)
	out := make([]Edge, 0)
	for _, e := range g.edges {
		if e.From == nodeID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].Type < out[j].Type
	})
	return out
}

// Incoming returns all incoming edges for a node.
func (g *Graph) Incoming(nodeID string) []Edge {
	if g == nil {
		return nil
	}
	nodeID = strings.TrimSpace(nodeID)
	out := make([]Edge, 0)
	for _, e := range g.edges {
		if e.To == nodeID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func edgeKey(e Edge) string {
	return strings.Join([]string{e.From, e.To, string(e.Type)}, "|")
}

func mergeNode(prev, next Node) Node {
	out := prev
	if out.Type == NodeTypeUnknown && next.Type != NodeTypeUnknown {
		out.Type = next.Type
	}
	if strings.TrimSpace(out.Label) == "" && strings.TrimSpace(next.Label) != "" {
		out.Label = strings.TrimSpace(next.Label)
	}
	if strings.TrimSpace(out.Jurisdiction) == "" && strings.TrimSpace(next.Jurisdiction) != "" {
		out.Jurisdiction = strings.TrimSpace(next.Jurisdiction)
	}
	out.Metadata = mergeMap(prev.Metadata, next.Metadata)
	return out
}

func mergeMap(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}
