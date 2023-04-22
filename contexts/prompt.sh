#!/bin/sh

pinocchio mine do \
    --context-file contexts/goal.txt \
    --context-file contexts/dsl.txt \
    --context-file contexts/tree-sitter-predicates-examples.txt \
    --instructions-file contexts/instructions/just-code.txt \
    --goal-file contexts/goals/parser.txt