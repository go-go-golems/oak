package guru

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Graph represents a simple directed graph of guru relationships.
type Graph struct {
	Nodes []GraphNode `json:"nodes" yaml:"nodes"`
	Edges []GraphEdge `json:"edges" yaml:"edges"`
}

// GraphNode describes a vertex in the graph.
type GraphNode struct {
	ID     string `json:"id" yaml:"id"`
	Label  string `json:"label" yaml:"label"`
	File   string `json:"file,omitempty" yaml:"file,omitempty"`
	Line   int    `json:"line,omitempty" yaml:"line,omitempty"`
	Column int    `json:"column,omitempty" yaml:"column,omitempty"`
	Kind   string `json:"kind,omitempty" yaml:"kind,omitempty"`
}

// GraphEdge connects two nodes.
type GraphEdge struct {
	From  string `json:"from" yaml:"from"`
	To    string `json:"to" yaml:"to"`
	Label string `json:"label,omitempty" yaml:"label,omitempty"`
}

var supportedGraphModes = map[string]struct{}{
	"referrers": {},
	"callees":   {},
	"callstack": {},
}

// BuildGraph constructs a graph for the provided guru results if the mode is supported.
func BuildGraph(mode, symbol string, pos *SymbolPosition, results []GuruResult) *Graph {
	mode = strings.ToLower(mode)
	if _, ok := supportedGraphModes[mode]; !ok {
		return nil
	}

	graph := &Graph{}
	nodeMap := map[string]GraphNode{}

	targetID := fmt.Sprintf("symbol-%s", SlugifyIdentifier(symbol))
	targetNode := GraphNode{
		ID:    targetID,
		Label: symbol,
		Kind:  "symbol",
	}
	if pos != nil {
		base := filepath.Base(pos.File)
		targetNode.File = pos.File
		targetNode.Line = pos.Line
		targetNode.Column = pos.Column
		targetNode.Label = fmt.Sprintf("%s (%s:%d)", symbol, base, pos.Line)
		targetNode.Kind = pos.Type
	}
	nodeMap[targetID] = targetNode

	switch mode {
	case "referrers":
		for _, result := range results {
			id, node := nodeFromResult(result, "referrer")
			nodeMap[id] = node
			graph.Edges = append(graph.Edges, GraphEdge{
				From:  id,
				To:    targetID,
				Label: limitText(result.Text),
			})
		}
	case "callees":
		for _, result := range results {
			id, node := nodeFromResult(result, "callee")
			nodeMap[id] = node
			graph.Edges = append(graph.Edges, GraphEdge{
				From:  targetID,
				To:    id,
				Label: limitText(result.Text),
			})
		}
	case "callstack":
		var prevID string
		for idx, result := range results {
			id, node := nodeFromResult(result, fmt.Sprintf("frame-%d", idx))
			nodeMap[id] = node
			if prevID != "" {
				graph.Edges = append(graph.Edges, GraphEdge{
					From: prevID,
					To:   id,
				})
			}
			prevID = id
		}
		if prevID != "" {
			graph.Edges = append(graph.Edges, GraphEdge{
				From: prevID,
				To:   targetID,
			})
		}
	}

	graph.Nodes = flattenNodeMap(nodeMap)
	return graph
}

// ToDOT renders the graph as Graphviz DOT.
func (g *Graph) ToDOT() string {
	var b strings.Builder
	b.WriteString("digraph G {\n")
	for _, node := range g.Nodes {
		label := escapeLabel(node.Label)
		b.WriteString(fmt.Sprintf("  %q [label=%q];\n", node.ID, label))
	}
	for _, edge := range g.Edges {
		if edge.Label != "" {
			b.WriteString(fmt.Sprintf("  %q -> %q [label=%q];\n", edge.From, edge.To, escapeLabel(edge.Label)))
		} else {
			b.WriteString(fmt.Sprintf("  %q -> %q;\n", edge.From, edge.To))
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// ToMermaid renders the graph using Mermaid syntax (left-to-right).
func (g *Graph) ToMermaid() string {
	var b strings.Builder
	b.WriteString("graph LR\n")
	for _, node := range g.Nodes {
		label := escapeLabel(node.Label)
		b.WriteString(fmt.Sprintf("  %s[%s]\n", node.ID, label))
	}
	for _, edge := range g.Edges {
		if edge.Label != "" {
			b.WriteString(fmt.Sprintf("  %s -- %s --> %s\n", edge.From, edge.Label, edge.To))
		} else {
			b.WriteString(fmt.Sprintf("  %s --> %s\n", edge.From, edge.To))
		}
	}
	return b.String()
}

// ToJSON renders the graph as JSON.
func (g *Graph) ToJSON() (string, error) {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func nodeFromResult(result GuruResult, kind string) (string, GraphNode) {
	label := fmt.Sprintf("%s:%d", filepath.Base(result.File), result.Line)
	id := fmt.Sprintf("%s-%d-%d", SlugifyIdentifier(result.File), result.Line, result.Column)
	if id == "" {
		id = fmt.Sprintf("node-%d-%d", result.Line, result.Column)
	}
	return id, GraphNode{
		ID:     id,
		Label:  label,
		File:   result.File,
		Line:   result.Line,
		Column: result.Column,
		Kind:   kind,
	}
}

func flattenNodeMap(nodes map[string]GraphNode) []GraphNode {
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := make([]GraphNode, 0, len(nodes))
	for _, k := range keys {
		res = append(res, nodes[k])
	}
	return res
}

func escapeLabel(label string) string {
	if label == "" {
		return label
	}
	label = strings.ReplaceAll(label, "\"", "'")
	return label
}

func limitText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > 60 {
		return text[:57] + "..."
	}
	return text
}

// SlugifyIdentifier turns any string into a filesystem/graph-friendly token.
func SlugifyIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var parts []string
	var current []rune

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, unicode.ToLower(r))
			continue
		}
		if len(current) > 0 {
			parts = append(parts, string(current))
			current = nil
		}
	}

	if len(current) > 0 {
		parts = append(parts, string(current))
	}

	return strings.Join(parts, "-")
}
