#!/bin/bash

set -e

cd "$(dirname "$0")"

echo "Building examples..."
cd function_finder && go build
cd ../typescript_analyzer && go build
cd ..

echo -e "\n============================================"
echo "Running Function Finder on Go example:"
echo -e "============================================\n"
./function_finder/function_finder ../../test-inputs/go-example.go

echo -e "\n============================================"
echo "Running TypeScript Analyzer on Component.tsx:"
echo -e "============================================\n"
./typescript_analyzer/typescript_analyzer ../../test-inputs/typescript/Component.tsx

echo -e "\n============================================"
echo "Running TypeScript Analyzer on App.tsx:"
echo -e "============================================\n"
./typescript_analyzer/typescript_analyzer ../../test-inputs/typescript/App.tsx