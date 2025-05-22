package tree_sitter

import (
	"io"

	"github.com/go-go-golems/oak/pkg/tree-sitter/dump"
	sitter "github.com/smacker/go-tree-sitter"
)

// DumpFormat represents a tree output format type
type DumpFormat = dump.Format

// DumpOptions contains settings for tree dumping
type DumpOptions = dump.Options

// Dumper is the interface that all tree dumpers must implement
type Dumper = dump.Dumper

// Format constants
const (
	FormatText DumpFormat = dump.FormatText
	FormatXML  DumpFormat = dump.FormatXML
	FormatJSON DumpFormat = dump.FormatJSON
	FormatYAML DumpFormat = dump.FormatYAML
)

// NewDumper creates a new tree dumper for the specified format
func NewDumper(format DumpFormat) Dumper {
	return dump.NewDumper(format)
}

// DumpTree dumps a tree to the specified writer using the given format and options
func DumpTree(tree *sitter.Tree, source []byte, w io.Writer, format DumpFormat, options DumpOptions) error {
	dumper := NewDumper(format)
	return dumper.Dump(tree, source, w, options)
}
