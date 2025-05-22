package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/go-go-golems/oak/pkg"
	"github.com/go-go-golems/oak/pkg/tree-sitter"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
)

// QueryBuilder provides an API for building and running tree-sitter queries
type QueryBuilder struct {
	language string
	queries  []Query
}

// Query represents a named tree-sitter query
type Query struct {
	Name  string
	Query string
}

// QueryOption is a functional option for configuring the QueryBuilder
type QueryOption func(*QueryBuilder)

// NewQueryBuilder creates a new query builder with the given options
func NewQueryBuilder(options ...QueryOption) *QueryBuilder {
	qb := &QueryBuilder{}

	for _, option := range options {
		option(qb)
	}

	return qb
}

// WithLanguage sets the language for the query builder
func WithLanguage(language string) QueryOption {
	return func(qb *QueryBuilder) {
		qb.language = language
	}
}

// WithQuery adds a query to the builder
func WithQuery(name, query string) QueryOption {
	return func(qb *QueryBuilder) {
		qb.queries = append(qb.queries, Query{Name: name, Query: query})
	}
}

// WithQueryFromFile adds a query from a file to the builder
func WithQueryFromFile(name, path string) QueryOption {
	return func(qb *QueryBuilder) {
		content, err := os.ReadFile(path)
		if err != nil {
			// Just log an error - we'll validate before running
			fmt.Printf("Error reading query file %s: %s\n", path, err)
			return
		}

		qb.queries = append(qb.queries, Query{Name: name, Query: string(content)})
	}
}

// FromYAML loads queries from a YAML file in the existing Oak format
func FromYAML(path string) QueryOption {
	return func(qb *QueryBuilder) {
		content, err := os.ReadFile(path)
		if err != nil {
			// Just log an error - we'll validate before running
			fmt.Printf("Error reading YAML file %s: %s\n", path, err)
			return
		}

		var yamlData struct {
			Language string                 `yaml:"language"`
			Queries  map[string]interface{} `yaml:"queries"`
		}

		err = yaml.Unmarshal(content, &yamlData)
		if err != nil {
			fmt.Printf("Error parsing YAML file %s: %s\n", path, err)
			return
		}

		if yamlData.Language != "" {
			qb.language = yamlData.Language
		}

		for name, query := range yamlData.Queries {
			queryStr, ok := query.(string)
			if ok {
				qb.queries = append(qb.queries, Query{Name: name, Query: queryStr})
			}
		}
	}
}

// RunConfig holds configuration for running queries
type RunConfig struct {
	Files      []string
	Glob       string
	Directory  string
	Recursive  bool
	MaxWorkers int
}

// RunOption is a functional option for configuring query execution
type RunOption func(*RunConfig)

// WithFiles specifies files to run queries on
func WithFiles(files []string) RunOption {
	return func(rc *RunConfig) {
		rc.Files = files
	}
}

// WithGlob specifies a glob pattern to find files
func WithGlob(pattern string) RunOption {
	return func(rc *RunConfig) {
		rc.Glob = pattern
	}
}

// WithDirectory specifies a directory to scan for files
func WithDirectory(dir string) RunOption {
	return func(rc *RunConfig) {
		rc.Directory = dir
	}
}

// WithRecursive enables recursive directory scanning
func WithRecursive(recursive bool) RunOption {
	return func(rc *RunConfig) {
		rc.Recursive = recursive
	}
}

// WithMaxWorkers sets the maximum number of worker goroutines
func WithMaxWorkers(n int) RunOption {
	return func(rc *RunConfig) {
		rc.MaxWorkers = n
	}
}

// QueryResults represents the raw query results by file
type QueryResults map[string]map[string]*tree_sitter.Result

// Run executes the queries and returns the raw results
func (qb *QueryBuilder) Run(
	ctx context.Context,
	options ...RunOption,
) (QueryResults, error) {
	if qb.language == "" {
		return nil, errors.New("language is required")
	}

	if len(qb.queries) == 0 {
		return nil, errors.New("at least one query is required")
	}

	config := &RunConfig{
		MaxWorkers: 4, // Default to 4 workers
	}

	for _, option := range options {
		option(config)
	}

	files, err := qb.resolveFiles(config)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("no files found to process")
	}

	// Convert our queries to tree-sitter format
	sitterQueries := make([]tree_sitter.SitterQuery, len(qb.queries))
	for i, q := range qb.queries {
		sitterQueries[i] = tree_sitter.SitterQuery{
			Name:  q.Name,
			Query: q.Query,
		}
	}

	// Get the tree-sitter language
	lang, err := qb.getLanguage()
	if err != nil {
		return nil, err
	}

	// Process files in parallel
	results := make(QueryResults)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	// Create a semaphore to limit concurrency
	sem := make(chan struct{}, config.MaxWorkers)

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("Error reading file %s: %s\n", file, err)
				return
			}

			// Create parser and parse the file
			parser := sitter.NewParser()
			parser.SetLanguage(lang)
			tree, err := parser.ParseCtx(ctx, nil, content)
			if err != nil {
				fmt.Printf("Error parsing file %s: %s\n", file, err)
				return
			}
			defer tree.Close()

			// Execute queries
			fileResults, err := tree_sitter.ExecuteQueries(lang, tree.RootNode(), sitterQueries, content)
			if err != nil {
				fmt.Printf("Error executing queries on file %s: %s\n", file, err)
				return
			}

			mutex.Lock()
			results[file] = fileResults
			mutex.Unlock()
		}(file)
	}

	wg.Wait()
	return results, nil
}

// getLanguage gets the tree-sitter language for the given language name
func (qb *QueryBuilder) getLanguage() (*sitter.Language, error) {
	// Use Oak's language utility
	return pkg.LanguageNameToSitterLanguage(qb.language)
}

// resolveFiles resolves the list of files to process based on the configuration
func (qb *QueryBuilder) resolveFiles(config *RunConfig) ([]string, error) {
	var files []string

	// Directly specified files
	if len(config.Files) > 0 {
		files = append(files, config.Files...)
	}

	// Glob pattern
	if config.Glob != "" {
		globFiles, err := filepath.Glob(config.Glob)
		if err != nil {
			return nil, errors.Wrap(err, "invalid glob pattern")
		}
		files = append(files, globFiles...)
	}

	// Directory scanning
	if config.Directory != "" {
		dirFiles, err := qb.scanDirectory(config.Directory, config.Recursive)
		if err != nil {
			return nil, err
		}
		files = append(files, dirFiles...)
	}

	// Deduplicate files
	deduped := make(map[string]struct{})
	var result []string

	for _, file := range files {
		if _, ok := deduped[file]; !ok {
			deduped[file] = struct{}{}
			result = append(result, file)
		}
	}

	return result, nil
}

// scanDirectory scans a directory for files
func (qb *QueryBuilder) scanDirectory(dir string, recursive bool) ([]string, error) {
	var files []string

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if path != dir && !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// TODO: Filter by language-specific extensions
		files = append(files, path)
		return nil
	}

	err := filepath.Walk(dir, walkFn)
	return files, err
}

// TemplatedResults is the data structure passed to templates
type TemplatedResults struct {
	Language      string
	ResultsByFile map[string]map[string]*tree_sitter.Result
}

// RunWithTemplate runs the queries and processes the results with a template
func (qb *QueryBuilder) RunWithTemplate(
	ctx context.Context,
	templateText string,
	options ...RunOption,
) (string, error) {
	results, err := qb.Run(ctx, options...)
	if err != nil {
		return "", err
	}

	// Create template data
	templateData := TemplatedResults{
		Language:      qb.language,
		ResultsByFile: results,
	}

	// Parse and execute template
	tmpl, err := template.New("query-results").Parse(templateText)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse template")
	}

	var output strings.Builder
	err = tmpl.Execute(&output, templateData)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute template")
	}

	return output.String(), nil
}

// RunWithTemplateFile runs the queries and processes the results with a template file
func (qb *QueryBuilder) RunWithTemplateFile(
	ctx context.Context,
	templatePath string,
	options ...RunOption,
) (string, error) {
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read template file")
	}

	return qb.RunWithTemplate(ctx, string(templateContent), options...)
}

// RunWithProcessor runs the queries and processes the results with a processor function
func (qb *QueryBuilder) RunWithProcessor(
	ctx context.Context,
	processor any,
	options ...RunOption,
) (any, error) {
	results, err := qb.Run(ctx, options...)
	if err != nil {
		return nil, err
	}

	// Type assertion for the processor function
	switch fn := processor.(type) {
	case func(QueryResults) (any, error):
		return fn(results)
	case func(QueryResults) ([]any, error):
		return fn(results)
	case func(QueryResults) (map[string]any, error):
		return fn(results)
	case func(QueryResults) (string, error):
		return fn(results)
	case func(QueryResults) (int, error):
		return fn(results)
	case func(QueryResults) (bool, error):
		return fn(results)
	default:
		return nil, errors.New("unsupported processor function type")
	}
}
