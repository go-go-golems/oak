#!/bin/sh

# tree-sitter parse contexts/test.go | sed -e 's/\[.*\] - \[.*\]//' -e 's/ )/)/'  | pbcopy

#pinocchio mine do \
#  --intro "Here is the tree-sitter query API documentation." \
#  --context-file contexts/golang-example.txt \
#  --context-file contexts/tree-sitter-query.md \
#  --context-file contexts/dsl.txt \
#  --goal-file  contexts/goals/spec.txt

#pinocchio mine do \
#  --intro "Here is the tree-sitter query API documentation." \
#  --context-file contexts/tree-sitter-query.md \
#  --context-file contexts/dsl.txt \
#  --goal "Write a YAML DSL file that covers all the examples in the tree-sitter query documentation."

pinocchio mine do \
  --context-file contexts/golang-example.txt \
  --goal "Rewrite the following YAML DSL file that contains tree-sitter queries to work against the GO AST." \
  --instructions-file queries/spec.yaml

#pinocchio mine do \
#   --context-file contexts/golang-example.txt \
#   --goal "extract all node types from the go tree-sitter AST and return as a bullet list."

pinocchio mine do \
     --context-file contexts/golang-types.txt \
     --goal "Use the go AST types to fix the tree-sitter queries in following YAML, because the node types often don't exist." \
     --instructions-file queries/go-spec.yaml

pinocchio mine do \
     --context-file contexts/golang-example.txt \
     --goal "Fix the tree-sitter queries in following YAML, because there are often missing intermediate nodes." \
     --instructions-file queries/go-spec-fixed.yaml

# go run ./cmd/oak run --query-file queries/go-spec-fixed2.yaml --input-file test-inputs/test.go 2>&1 | grep level | jq -r ".query"

pinocchio mine do \
     --context-file queries/go-spec-fixed2.yaml \
     --goal "Use the tree-sitter queries listed above to generate a go file covering all the cases."