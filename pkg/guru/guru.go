package guru

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GuruResult represents a guru query result
type GuruResult struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Text   string `json:"text"`
	Kind   string `json:"kind,omitempty"`
}

// RunGuruQuery executes guru command with given position and returns results
func RunGuruQuery(mode, position string, jsonOutput bool) ([]GuruResult, error) {
	args := []string{mode, position}
	if jsonOutput {
		args = append([]string{"-json"}, args...)
	}

	cmd := exec.Command("guru", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("guru command failed: %w", err)
	}

	if jsonOutput {
		return parseJSONOutput(output)
	}

	return parseTextOutput(output), nil
}

// parseJSONOutput parses guru JSON output
func parseJSONOutput(output []byte) ([]GuruResult, error) {
	var results []GuruResult

	// Guru JSON output is line-delimited JSON
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var result GuruResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue // Skip invalid JSON lines
		}
		results = append(results, result)
	}

	return results, nil
}

// parseTextOutput parses guru text output (pos: text format)
func parseTextOutput(output []byte) []GuruResult {
	var results []GuruResult
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse "filename:line:col: text" format
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 3 {
			result := GuruResult{
				Text: strings.TrimSpace(parts[len(parts)-1]),
			}

			// Try to parse line and column
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &result.Line)
			}
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &result.Column)
			}
			if len(parts) >= 1 {
				result.File = parts[0]
			}

			results = append(results, result)
		} else {
			// Handle "-: text" format (unknown position)
			result := GuruResult{
				File: "-",
				Text: line,
			}
			results = append(results, result)
		}
	}

	return results
}

