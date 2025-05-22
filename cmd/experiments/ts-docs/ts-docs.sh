#!/bin/bash

# This script runs the ts-docs tool and writes the output to a markdown file

if [ $# -lt 1 ]; then
    echo "Usage: $0 <file.ts/directory> [output.md]"
    echo "If output.md is not specified, it will use the basename of the input file/directory + -api.md"
    exit 1
fi

INPUT="$1"

# Determine output filename
if [ $# -ge 2 ]; then
    OUTPUT="$2"
else
    # Default output file based on input name
    BASENAME=$(basename "$INPUT")
    FILENAME="${BASENAME%.*}"
    OUTPUT="${FILENAME}-api.md"
fi

# Run the tool and save output
echo "Generating API documentation for $INPUT..."
go run main.go "$INPUT" >"$OUTPUT"

if [ $? -eq 0 ]; then
    echo "API documentation written to $OUTPUT"
    echo "Word count: $(wc -w <"$OUTPUT") words"
    echo "Function count: $(grep -c '^###' "$OUTPUT") functions"
else
    echo "Error generating documentation"
    exit 1
fi
