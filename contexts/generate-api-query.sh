#!/bin/sh

pinocchio mine do \
  --intro "Here is the tree-sitter query API documentation." \
  --context-file contexts/tree-sitter-query.md \
  --context-file contexts/dsl.txt \
  --context-file contexts/golang-example.txt \
  --goal "I want to generate a DSL YAML file that covers all cases covered in the documentation, against a go AST."