package dump

import (
	"io"

	sitter "github.com/smacker/go-tree-sitter"
)

// Format represents a tree output format type
type Format string

const (
	// FormatText is the enhanced text output format
	FormatText Format = "text"
	// FormatXML is the XML output format
	FormatXML Format = "xml"
	// FormatJSON is the JSON output format
	FormatJSON Format = "json"
	// FormatYAML is the YAML output format
	FormatYAML Format = "yaml"
)

// Options contains settings for tree dumping
type Options struct {
	ShowBytes      bool
	ShowContent    bool
	ShowAttributes bool
	SkipWhitespace bool
}

// Dumper is the interface that all tree dumpers must implement
type Dumper interface {
	Dump(tree *sitter.Tree, source []byte, w io.Writer, options Options) error
}

// NewDumper creates a new tree dumper for the specified format
func NewDumper(format Format) Dumper {
	switch format {
	case FormatText:
		return &TextDumper{}
	case FormatXML:
		return &XMLDumper{}
	case FormatJSON:
		return &JSONDumper{}
	case FormatYAML:
		return &YAMLDumper{}
	default:
		return &TextDumper{} // Default to text format
	}
}
