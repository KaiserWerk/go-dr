package graphrag

import (
	"bytes"
	"encoding/json"
	"io"
)

// GraphSnapshot is a deterministic serializable graph view.
type GraphSnapshot struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Snapshot returns a deterministic serializable graph view.
func Snapshot(g *Graph) GraphSnapshot {
	if g == nil {
		return GraphSnapshot{}
	}
	return GraphSnapshot{
		Nodes: g.Nodes(),
		Edges: g.Edges(),
	}
}

// MarshalGraph encodes a graph snapshot to pretty JSON.
func MarshalGraph(g *Graph) ([]byte, error) {
	return json.MarshalIndent(Snapshot(g), "", "  ")
}

// WriteGraph writes one graph snapshot as pretty JSON.
func WriteGraph(w io.Writer, g *Graph) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(Snapshot(g))
}

// MarshalNodesJSONL encodes graph nodes in JSONL format.
func MarshalNodesJSONL(g *Graph) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := WriteNodesJSONL(buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteNodesJSONL writes one compact JSON object per node.
func WriteNodesJSONL(w io.Writer, g *Graph) error {
	enc := json.NewEncoder(w)
	for _, node := range Snapshot(g).Nodes {
		if err := enc.Encode(node); err != nil {
			return err
		}
	}
	return nil
}

// MarshalEdgesJSONL encodes graph edges in JSONL format.
func MarshalEdgesJSONL(g *Graph) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := WriteEdgesJSONL(buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteEdgesJSONL writes one compact JSON object per edge.
func WriteEdgesJSONL(w io.Writer, g *Graph) error {
	enc := json.NewEncoder(w)
	for _, edge := range Snapshot(g).Edges {
		if err := enc.Encode(edge); err != nil {
			return err
		}
	}
	return nil
}
