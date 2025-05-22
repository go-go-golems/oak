# Proposals for Enhanced DumpTree Output Format

This document outlines several proposals to improve the `DumpTree` function in the Oak command package, making the output more comprehensive and useful for debugging and analysis.

## Current Implementation

The current `DumpTree` implementation in `oak/pkg/cmds/cmd.go` outputs a simple indented text representation of the tree structure:

```
source_file [0-29]
    function_declaration [0-29]
        func [0-4]
        name: identifier [5-6]
        parameters: parameter_list [6-29]
            ( [6-7]
            parameter_declaration [7-18]
                name: identifier [7-8]
                , [8-9]
                name: identifier [10-11]
                , [11-12]
                name: identifier [13-14]
                type: type_identifier [15-18]
```

While this format shows the hierarchical structure of the syntax tree, it has several limitations:
- It doesn't show the actual content of nodes (the source code text)
- It doesn't expose all node attributes available from the tree-sitter API
- It doesn't provide machine-readable output that could be easily parsed or analyzed programmatically
- It doesn't allow filtering or focusing on specific parts of the tree

## Proposal 1: XML Output Format

### Overview
XML provides a hierarchical format that naturally maps to the tree structure and supports attributes for node metadata.

### Example Implementation

```go
func (oc *OakCommand) DumpTreeXML(tree *sitter.Tree, source []byte, w io.Writer, options map[string]bool) error {
    showBytes := false
    if options != nil {
        if val, ok := options["showBytes"]; ok {
            showBytes = val
        }
    }
    
    fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
    fmt.Fprintf(w, "<tree>\n")
    
    var visitXML func(n *sitter.Node, depth int) error
    visitXML = func(n *sitter.Node, depth int) error {
        if n.IsNull() {
            return nil
        }
        
        indent := strings.Repeat("  ", depth)
        nodeType := n.Type()
        
        // Skip pure whitespace nodes
        if matched, _ := regexp.MatchString(`^\s+$`, nodeType); matched {
            return nil
        }
        
        // Get node text content if source is provided
        var contentAttr string
        if source != nil {
            content := n.Content(source)
            // XML-escape the content
            content = strings.ReplaceAll(content, "&", "&amp;")
            content = strings.ReplaceAll(content, "<", "&lt;")
            content = strings.ReplaceAll(content, ">", "&gt;")
            content = strings.ReplaceAll(content, "\"", "&quot;")
            contentAttr = fmt.Sprintf(" content=\"%s\"", content)
        }
        
        // Convert to 1-based line/column numbers for better readability
        startPoint := n.StartPoint()
        endPoint := n.EndPoint()
        startLine := startPoint.Row + 1
        startCol := startPoint.Column + 1
        endLine := endPoint.Row + 1
        endCol := endPoint.Column + 1
        
        // Format position as "startLine,startCol-endLine,endCol"
        posAttr := fmt.Sprintf(" pos=\"%d,%d-%d,%d\"", startLine, startCol, endLine, endCol)
        
        // Optionally include byte positions
        bytesAttr := ""
        if showBytes {
            bytesAttr = fmt.Sprintf(" bytes=\"%d-%d\"", n.StartByte(), n.EndByte())
        }
        
        // Output opening tag with attributes
        fmt.Fprintf(w, "%s<node type=\"%s\"%s%s named=\"%t\" missing=\"%t\" extra=\"%t\" has_error=\"%t\"%s>\n",
            indent, 
            nodeType,
            posAttr,
            bytesAttr,
            n.IsNamed(),
            n.IsMissing(),
            n.IsExtra(),
            n.HasError(),
            contentAttr,
        )
        
        // Visit children
        for i := 0; i < int(n.ChildCount()); i++ {
            fieldName := n.FieldNameForChild(i)
            child := n.Child(i)
            
            if fieldName != "" {
                fmt.Fprintf(w, "%s  <field name=\"%s\">\n", indent, fieldName)
                err := visitXML(child, depth+2)
                if err != nil {
                    return err
                }
                fmt.Fprintf(w, "%s  </field>\n", indent)
            } else {
                err := visitXML(child, depth+1)
                if err != nil {
                    return err
                }
            }
        }
        
        // Output closing tag
        fmt.Fprintf(w, "%s</node>\n", indent)
        return nil
    }
    
    err := visitXML(tree.RootNode(), 1)
    if err != nil {
        return err
    }
    
    fmt.Fprintf(w, "</tree>\n")
    return nil
}
```

### Example Output

```xml
<?xml version="1.0" encoding="UTF-8"?>
<tree>
  <node type="source_file" pos="1,1-3,2" bytes="0-29" named="true" missing="false" extra="false" has_error="false" content="func a(b, c, d int) {}">
    <node type="function_declaration" pos="1,1-3,2" named="true" missing="false" extra="false" has_error="false" content="func a(b, c, d int) {}">
      <node type="func" pos="1,1-1,5" named="false" missing="false" extra="false" has_error="false" content="func"></node>
      <field name="name">
        <node type="identifier" pos="1,6-1,7" named="true" missing="false" extra="false" has_error="false" content="a"></node>
      </field>
      <field name="parameters">
        <node type="parameter_list" pos="1,7-1,19" named="true" missing="false" extra="false" has_error="false" content="(b, c, d int)">
          <node type="(" pos="1,7-1,8" named="false" missing="false" extra="false" has_error="false" content="("></node>
          <node type="parameter_declaration" pos="1,8-1,18" named="true" missing="false" extra="false" has_error="false" content="b, c, d int">
            <field name="name">
              <node type="identifier" pos="1,8-1,9" named="true" missing="false" extra="false" has_error="false" content="b"></node>
            </field>
            <node type="," pos="1,9-1,10" named="false" missing="false" extra="false" has_error="false" content=","></node>
            <field name="name">
              <node type="identifier" pos="1,11-1,12" named="true" missing="false" extra="false" has_error="false" content="c"></node>
            </field>
            <node type="," pos="1,12-1,13" named="false" missing="false" extra="false" has_error="false" content=","></node>
            <field name="name">
              <node type="identifier" pos="1,14-1,15" named="true" missing="false" extra="false" has_error="false" content="d"></node>
            </field>
            <field name="type">
              <node type="type_identifier" pos="1,16-1,19" named="true" missing="false" extra="false" has_error="false" content="int"></node>
            </field>
          </node>
          <node type=")" pos="1,18-1,19" named="false" missing="false" extra="false" has_error="false" content=")"></node>
        </node>
      </field>
      <field name="body">
        <node type="block" pos="1,20-3,2" named="true" missing="false" extra="false" has_error="false" content=" {}">
          <node type="{" pos="1,20-1,21" named="false" missing="false" extra="false" has_error="false" content="{"></node>
          <node type="}" pos="3,1-3,2" named="false" missing="false" extra="false" has_error="false" content="}"></node>
        </node>
      </field>
    </node>
  </node>
</tree>
```

### Benefits
- Highly structured and machine-readable
- Uses compact position format "line,col-line,col" for consistency
- Optional byte offset representation
- Includes all node attributes
- Properly handles field names
- Includes source text when available
- Can be parsed with standard XML tools
- Good for detailed debugging and analysis

### Limitations
- Verbose output may be too detailed for quick browsing
- Requires XML parsing to process programmatically

## Proposal 2: JSON Output Format

### Overview
JSON format provides a more compact, modern alternative that is easy to parse in many programming languages.

### Example Implementation

```go
func (oc *OakCommand) DumpTreeJSON(tree *sitter.Tree, source []byte, w io.Writer, options map[string]bool) error {
    showBytes := false
    if options != nil {
        if val, ok := options["showBytes"]; ok {
            showBytes = val
        }
    }
    
    type NodeJSON struct {
        Type      string     `json:"type"`
        Position  string     `json:"pos"`  // Format: "startLine,startCol-endLine,endCol"
        Bytes     string     `json:"bytes,omitempty"` // Optional
        IsNamed   bool       `json:"is_named,omitempty"`
        IsMissing bool       `json:"is_missing,omitempty"`
        IsExtra   bool       `json:"is_extra,omitempty"`
        HasError  bool       `json:"has_error,omitempty"`
        Content   string     `json:"content,omitempty"`
        Fields    map[string][]*NodeJSON `json:"fields,omitempty"`
        Children  []*NodeJSON `json:"children,omitempty"`
    }
    
    var buildJSON func(n *sitter.Node) *NodeJSON
    buildJSON = func(n *sitter.Node) *NodeJSON {
        if n.IsNull() {
            return nil
        }
        
        nodeType := n.Type()
        // Skip pure whitespace nodes
        if matched, _ := regexp.MatchString(`^\s+$`, nodeType); matched {
            return nil
        }
        
        // Convert to 1-based line/column numbers for better readability
        startPoint := n.StartPoint()
        endPoint := n.EndPoint()
        startLine := startPoint.Row + 1
        startCol := startPoint.Column + 1
        endLine := endPoint.Row + 1
        endCol := endPoint.Column + 1
        
        // Format position as "startLine,startCol-endLine,endCol"
        position := fmt.Sprintf("%d,%d-%d,%d", startLine, startCol, endLine, endCol)
        
        // Create the JSON node
        node := &NodeJSON{
            Type:     nodeType,
            Position: position,
        }
        
        // Add byte range if requested
        if showBytes {
            node.Bytes = fmt.Sprintf("%d-%d", n.StartByte(), n.EndByte())
        }
        
        // Only include these fields if true
        if n.IsNamed() {
            node.IsNamed = true
        }
        if n.IsMissing() {
            node.IsMissing = true
        }
        if n.IsExtra() {
            node.IsExtra = true
        }
        if n.HasError() {
            node.HasError = true
        }
        
        // Add content if source is provided
        if source != nil {
            content := n.Content(source)
            // For very long content, truncate
            if len(content) > 60 {
                content = content[:57] + "..."
            }
            node.Content = content
        }
        
        // Process children
        fieldMap := make(map[string][]*NodeJSON)
        var plainChildren []*NodeJSON
        
        for i := 0; i < int(n.ChildCount()); i++ {
            fieldName := n.FieldNameForChild(i)
            child := buildJSON(n.Child(i))
            
            if child == nil {
                continue
            }
            
            if fieldName != "" {
                if fieldMap[fieldName] == nil {
                    fieldMap[fieldName] = []*NodeJSON{}
                }
                fieldMap[fieldName] = append(fieldMap[fieldName], child)
            } else {
                plainChildren = append(plainChildren, child)
            }
        }
        
        if len(fieldMap) > 0 {
            node.Fields = fieldMap
        }
        
        if len(plainChildren) > 0 {
            node.Children = plainChildren
        }
        
        return node
    }
    
    rootJSON := buildJSON(tree.RootNode())
    if rootJSON == nil {
        return errors.New("failed to build JSON representation of tree")
    }
    
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    return enc.Encode(rootJSON)
}
```

### Example Output

```json
{
  "type": "source_file",
  "pos": "1,1-3,2",
  "bytes": "0-29",
  "is_named": true,
  "content": "func a(b, c, d int) {}",
  "children": [
    {
      "type": "function_declaration",
      "pos": "1,1-3,2",
      "is_named": true,
      "content": "func a(b, c, d int) {}",
      "children": [
        {
          "type": "func",
          "pos": "1,1-1,5",
          "content": "func"
        }
      ],
      "fields": {
        "name": [
          {
            "type": "identifier",
            "pos": "1,6-1,7",
            "is_named": true,
            "content": "a"
          }
        ],
        "parameters": [
          {
            "type": "parameter_list",
            "pos": "1,7-1,19",
            "is_named": true,
            "content": "(b, c, d int)",
            "children": [
              {
                "type": "(",
                "pos": "1,7-1,8",
                "content": "("
              },
              {
                "type": ")",
                "pos": "1,18-1,19",
                "content": ")"
              }
            ],
            "fields": {
              "parameter_declaration": [
                {
                  "type": "parameter_declaration",
                  "pos": "1,8-1,18",
                  "is_named": true,
                  "content": "b, c, d int",
                  "children": [
                    {
                      "type": ",",
                      "pos": "1,9-1,10",
                      "content": ","
                    },
                    {
                      "type": ",",
                      "pos": "1,12-1,13",
                      "content": ","
                    }
                  ],
                  "fields": {
                    "name": [
                      {
                        "type": "identifier",
                        "pos": "1,8-1,9",
                        "is_named": true,
                        "content": "b"
                      },
                      {
                        "type": "identifier",
                        "pos": "1,11-1,12",
                        "is_named": true,
                        "content": "c"
                      },
                      {
                        "type": "identifier",
                        "pos": "1,14-1,15",
                        "is_named": true,
                        "content": "d"
                      }
                    ],
                    "type": [
                      {
                        "type": "type_identifier",
                        "pos": "1,16-1,19",
                        "is_named": true,
                        "content": "int"
                      }
                    ]
                  }
                }
              ]
            }
          }
        ],
        "body": [
          {
            "type": "block",
            "pos": "1,20-3,2",
            "is_named": true,
            "content": " {}",
            "children": [
              {
                "type": "{",
                "pos": "1,20-1,21",
                "content": "{"
              },
              {
                "type": "}",
                "pos": "3,1-3,2",
                "content": "}"
              }
            ]
          }
        ]
      }
    }
  ]
}
```

### Benefits
- Compact but still comprehensive representation
- Uses consistent position format with other formats
- Optional byte position information
- Native format for JavaScript and web-based tools
- Easy to process programmatically in most languages
- Supports deeply nested structures

### Limitations
- Can be less human-readable for very large trees
- Requires a JSON parser to process

## Proposal 3: Enhanced Text Format with Annotations

### Overview
An improved text-based format that includes more information while maintaining readability.

### Example Implementation

```go
func (oc *OakCommand) DumpTreeEnhanced(tree *sitter.Tree, source []byte, w io.Writer, options map[string]bool) error {
    // Default options
    showContent := true
    showAttributes := true
    skipWhitespace := true
    showBytes := false
    
    if options != nil {
        if val, ok := options["showContent"]; ok {
            showContent = val
        }
        if val, ok := options["showAttributes"]; ok {
            showAttributes = val
        }
        if val, ok := options["skipWhitespace"]; ok {
            skipWhitespace = val
        }
        if val, ok := options["showBytes"]; ok {
            showBytes = val
        }
    }
    
    var visitEnhanced func(n *sitter.Node, name string, depth int) error
    visitEnhanced = func(n *sitter.Node, name string, depth int) error {
        if n.IsNull() {
            return nil
        }
        
        nodeType := n.Type()
        // Skip whitespace nodes if configured
        if skipWhitespace {
            if matched, _ := regexp.MatchString(`^\s+$`, nodeType); matched {
                return nil
            }
        }
        
        indent := strings.Repeat("  ", depth)
        prefix := ""
        if name != "" {
            prefix = name + ": "
        }
        
        // Convert to 1-based line/column numbers for better readability
        startPoint := n.StartPoint()
        endPoint := n.EndPoint()
        startLine := startPoint.Row + 1
        startCol := startPoint.Column + 1
        endLine := endPoint.Row + 1
        endCol := endPoint.Column + 1
        
        // Format position as "startLine,startCol-endLine,endCol"
        position := fmt.Sprintf("%d,%d-%d,%d", startLine, startCol, endLine, endCol)
        
        // Basic node info with position
        fmt.Fprintf(w, "%s%s%s [%s]", indent, prefix, nodeType, position)
        
        // Add byte offsets if requested
        if showBytes {
            fmt.Fprintf(w, " bytes:%d-%d", n.StartByte(), n.EndByte())
        }
        
        // Additional attributes if requested
        if showAttributes {
            flags := []string{}
            if n.IsNamed() {
                flags = append(flags, "named")
            }
            if n.IsMissing() {
                flags = append(flags, "missing")
            }
            if n.IsExtra() {
                flags = append(flags, "extra")
            }
            if n.HasError() {
                flags = append(flags, "error")
            }
            
            if len(flags) > 0 {
                fmt.Fprintf(w, " [%s]", strings.Join(flags, ","))
            }
        }
        
        // Content if source is provided and requested
        if source != nil && showContent {
            content := n.Content(source)
            // Escape newlines and other control characters for display
            content = strings.ReplaceAll(content, "\n", "\\n")
            content = strings.ReplaceAll(content, "\t", "\\t")
            content = strings.ReplaceAll(content, "\r", "\\r")
            
            // Truncate very long content
            if len(content) > 60 {
                content = content[:57] + "..."
            }
            
            fmt.Fprintf(w, " \"%s\"", content)
        }
        
        fmt.Fprintln(w)
        
        // Visit children
        for i := 0; i < int(n.ChildCount()); i++ {
            fieldName := n.FieldNameForChild(i)
            err := visitEnhanced(n.Child(i), fieldName, depth+1)
            if err != nil {
                return err
            }
        }
        
        return nil
    }
    
    return visitEnhanced(tree.RootNode(), "", 0)
}
```

### Example Output

```
source_file [1,1-3,2] bytes:0-29 [named] "func a(b, c, d int) {}"
  function_declaration [1,1-3,2] [named] "func a(b, c, d int) {}"
    func [1,1-1,5] "func"
    name: identifier [1,6-1,7] [named] "a"
    parameters: parameter_list [1,7-1,19] [named] "(b, c, d int)"
      ( [1,7-1,8] "("
      parameter_declaration [1,8-1,18] [named] "b, c, d int"
        name: identifier [1,8-1,9] [named] "b"
        , [1,9-1,10] ","
        name: identifier [1,11-1,12] [named] "c"
        , [1,12-1,13] ","
        name: identifier [1,14-1,15] [named] "d"
        type: type_identifier [1,16-1,19] [named] "int"
      ) [1,18-1,19] ")"
    body: block [1,20-3,2] [named] " {}"
      { [1,20-1,21] "{"
      } [3,1-3,2] "}"
```

### Benefits
- Human-readable format similar to the current implementation
- Uses consistent position format with other formats (line,col-line,col)
- Optional byte offsets when needed for lower-level analysis
- Includes more detailed information like attributes and content
- Customizable output with options to show/hide different details
- Does not require special parsers to read

### Limitations
- Not as machine-friendly as XML, JSON, or YAML
- Can still be verbose for large trees
- Limited formatting options for complex structures

## Proposal 4: YAML Output Format with Line/Character Positions

### Overview
YAML provides a clean, human-readable hierarchical format that is also machine-parseable, with the benefit of being more compact than XML and more readable than JSON for complex structures. This proposal focuses on using line and character positions rather than byte offsets.

### Example Implementation

```go
func (oc *OakCommand) DumpTreeYAML(tree *sitter.Tree, source []byte, w io.Writer, options map[string]bool) error {
    showBytes := false
    if options != nil {
        if val, ok := options["showBytes"]; ok {
            showBytes = val
        }
    }
    
    type NodeYAML struct {
        Type     string `yaml:"type"`
        Position string `yaml:"pos"`      // Format: "startLine,startCol-endLine,endCol"
        Bytes    string `yaml:"bytes,omitempty"` // Format: "startByte-endByte" (optional)
        Named    bool   `yaml:"named,omitempty"`
        Missing  bool   `yaml:"missing,omitempty"`
        Extra    bool   `yaml:"extra,omitempty"`
        HasError bool   `yaml:"has_error,omitempty"`
        Content  string `yaml:"content,omitempty"`
        Fields   map[string][]*NodeYAML `yaml:"fields,omitempty"`
        Children []*NodeYAML `yaml:"children,omitempty"`
    }
    
    var buildYAML func(n *sitter.Node) *NodeYAML
    buildYAML = func(n *sitter.Node) *NodeYAML {
        if n.IsNull() {
            return nil
        }
        
        nodeType := n.Type()
        // Skip pure whitespace nodes
        if matched, _ := regexp.MatchString(`^\s+$`, nodeType); matched {
            return nil
        }
        
        // Create the YAML node using compact line/column position format
        startPoint := n.StartPoint()
        endPoint := n.EndPoint()
        
        // Convert to 1-based line/column numbers for better readability
        startLine := startPoint.Row + 1 
        startCol := startPoint.Column + 1
        endLine := endPoint.Row + 1
        endCol := endPoint.Column + 1
        
        // Format: "startLine,startCol-endLine,endCol"
        position := fmt.Sprintf("%d,%d-%d,%d", startLine, startCol, endLine, endCol)
        
        node := &NodeYAML{
            Type:     nodeType,
            Position: position,
        }
        
        // Add byte range if requested
        if showBytes {
            node.Bytes = fmt.Sprintf("%d-%d", n.StartByte(), n.EndByte())
        }
        
        // Only include these fields if true to keep output clean
        if n.IsNamed() {
            node.Named = true
        }
        if n.IsMissing() {
            node.Missing = true
        }
        if n.IsExtra() {
            node.Extra = true
        }
        if n.HasError() {
            node.HasError = true
        }
        
        // Add content if source is provided
        if source != nil {
            content := n.Content(source)
            // Clean up content for YAML
            content = strings.ReplaceAll(content, "\n", "\\n")
            content = strings.ReplaceAll(content, "\t", "\\t")
            content = strings.ReplaceAll(content, "\r", "\\r")
            
            // Truncate very long content
            if len(content) > 60 {
                content = content[:57] + "..."
            }
            
            node.Content = content
        }
        
        // Process children
        fieldMap := make(map[string][]*NodeYAML)
        var plainChildren []*NodeYAML
        
        for i := 0; i < int(n.ChildCount()); i++ {
            fieldName := n.FieldNameForChild(i)
            child := buildYAML(n.Child(i))
            
            if child == nil {
                continue
            }
            
            if fieldName != "" {
                if fieldMap[fieldName] == nil {
                    fieldMap[fieldName] = []*NodeYAML{}
                }
                fieldMap[fieldName] = append(fieldMap[fieldName], child)
            } else {
                plainChildren = append(plainChildren, child)
            }
        }
        
        if len(fieldMap) > 0 {
            node.Fields = fieldMap
        }
        
        if len(plainChildren) > 0 {
            node.Children = plainChildren
        }
        
        return node
    }
    
    rootYAML := buildYAML(tree.RootNode())
    if rootYAML == nil {
        return errors.New("failed to build YAML representation of tree")
    }
    
    enc := yaml.NewEncoder(w)
    defer enc.Close()
    return enc.Encode(rootYAML)
}
```

### Example Output

```yaml
type: source_file
pos: 1,1-3,2
bytes: 0-29  # Optional, shown with showBytes=true
named: true
content: "func a(b, c, d int) {}"
children:
  - type: function_declaration
    pos: 1,1-3,2
    named: true
    content: "func a(b, c, d int) {}"
    children:
      - type: func
        pos: 1,1-1,5
        content: "func"
    fields:
      name:
        - type: identifier
          pos: 1,6-1,7
          named: true
          content: "a"
      parameters:
        - type: parameter_list
          pos: 1,7-1,19
          named: true
          content: "(b, c, d int)"
          children:
            - type: (
              pos: 1,7-1,8
              content: "("
            - type: )
              pos: 1,18-1,19
              content: ")"
          fields:
            parameter_declaration:
              - type: parameter_declaration
                pos: 1,8-1,18
                named: true
                content: "b, c, d int"
                children:
                  - type: ","
                    pos: 1,9-1,10
                    content: ","
                  - type: ","
                    pos: 1,12-1,13
                    content: ","
                fields:
                  name:
                    - type: identifier
                      pos: 1,8-1,9
                      named: true
                      content: "b"
                    - type: identifier
                      pos: 1,11-1,12
                      named: true
                      content: "c"
                    - type: identifier
                      pos: 1,14-1,15
                      named: true
                      content: "d"
                  type:
                    - type: type_identifier
                      pos: 1,16-1,19
                      named: true
                      content: "int"
      body:
        - type: block
          pos: 1,20-3,2
          named: true
          content: " {}"
          children:
            - type: "{"
              pos: 1,20-1,21
              content: "{"
            - type: "}"
              pos: 3,1-3,2
              content: "}"
```

### Benefits
- **More compact position representation** using the format "line,col-line,col"
- Optional byte range representation for lower-level analysis
- Clean, hierarchical representation that is both human and machine-readable
- 1-based line/column numbering that aligns with most editor conventions
- Compact format with minimal punctuation overhead
- Excellent for both inspection and programmatic processing
- Familiar format for developers working with configuration files

### Limitations
- Slightly less compact than JSON for dense trees
- Indentation-based format can be problematic for very deep trees
- YAML parsers are not as universally available as JSON parsers

## Updated Recommendation

I recommend implementing all four output formats with the consistent position representation format, with YAML as the primary format:

1. **YAML Format** (primary): Clean, readable, with compact position representation
2. **XML Format**: For integration with XML-based tools
3. **JSON Format**: For web-based visualization
4. **Enhanced Text Format**: For quick human inspection

The command-line parameters would be consistent across all formats:

```
--dump-format=yaml|xml|json|text
--show-bytes=true|false    # For all formats, show byte offsets in addition to line/col
--show-content=true|false  # Show source text content in nodes
--show-attributes=true|false  # Show node attributes like "named", "missing", etc.
--skip-whitespace=true|false  # Skip whitespace-only nodes
```

The YAML format should be the default when no format is specified. 