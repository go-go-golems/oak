---
Title: Create a source query with oak
Slug: create-query
Topics:
- oak
- query
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

## Creating a query file

To create a query verb with oak, you need to create a YAML file that 
provides a high-level [glaze](https://github.com/go-go-golems/glazed) Command
description consisting of:

- name (the verb used to run your query)
- short (a short description of the verb)
- long (optional, a long form description of the verb)
- flags (a list of glaze.Command flags description the command-line flags the verb provides)

Arguments to the verb are a list of source files. 

Furthermore, the YAML file should provide the following oak specific fields:

- language (the name of the grammar to be used)
- queries (a list of queries that have two fields: name and query)
- template (the template used to render the results)

This file should be stored in one of the repositories configured for oak (by default, `~/.oak/queries`).
The subdirectory the file is in will be translated to subverbs. For example, a file in `~/.oak/queries/foo/bar/bla.yaml`
will be mapped to the verb `oak foo bar bla`.

Further "repositories" (directories containing query files) can be added by editing the configuration file
`~/.oak/config.yaml` and adding a `repositories` field (a simple list of locations).

## Example query

Here is a simple query file that prints function declarations and import statements for go files.

```yaml
name: example1
short: A simple example to extract go imports and functions from a go file

flags:
  - name: verbose
    type: bool
    help: Output all results
    default: false

language: go
queries:
  - name: functionDeclarations
    query: |
      (function_declaration
        name: (identifier) @name
        parameters: (parameter_list) @parameters
        body: (block))
  - name: importStatements
    query: |
      (import_declaration
        (import_spec_list [
          (import_spec
            (package_identifier) @name
             path: (interpreted_string_literal) @path)
          (import_spec
            path: (interpreted_string_literal) @path)
        ]))

template: |
  {{ range $file, $results := .ResultsByFile -}}
  File: {{ $file }}

  {{ with $results -}}
  Function Declarations:
  {{- range .functionDeclarations.Matches }}
  - {{ .name.Text }}{{ .parameters.Text }} {{ end }}

  Import Statements:
  {{ range .importStatements.Matches -}}
  - {{ if .name }}name: {{ .name.Text }}, {{end -}} path: {{ .path.Text }}
  {{ end }}
  {{ end -}}
  {{ end -}}

  {{ if .verbose -}}
  Results:{{ range $v := .Results }}
    {{ $v.Name }}: {{ range $match := $v.Matches }}{{ range $captureName, $captureValue := $match }}
       {{ $captureName }}: {{ $captureValue.Text }}{{ end }}
    {{end}}{{ end }}
  {{ end -}}
```

## Command execution

To call the command, run `oak` with the verb path given by the subdirectory structure of the command location
within its repository directory. The verb accepts the flags defined in the YAML file as well as a few additional ones
preconfigured by oak and glaze. To get a full list of accepted flags, run `oak CMD --help`:

```
❯ oak example1 --help

   example1 - A simple example to extract go imports and functions from a go  
  file                                                                        
                                                                              
  For more help, run:  oak help example1                                      
                                                                              
  ## Usage:                                                                   
                                                                              
   oak example1 [flags]                                                       
                                                                              
  ## Flags:                                                                   
                                                                              
           --create-alias    Create a CLI alias for the query                 
       --create-cliopatra    Print the CLIopatra YAML for the command         
         --create-command    Create a new command for the query, with the     
  defaults updated                                                            
               -h, --help    help for example1                                
                --verbose    Output all results                               
                                                                              
  ## Global flags:                                                            
                                                                              
                 --config    Path to config file (default ~/.oak/config.yml)  
               --log-file    Log file (default: stderr)                       
             --log-format    Log format (json, text) (default "text")         
              --log-level    Log level (debug, info, warn, error, fatal) (default
  "info")                                                                     
            --with-caller    Log caller                                       
```

oak will then:
- iterate over the given input files
- parse each one using treesitter and the configured language grammar into a new AST
- run each query against that AST
- the query results are stored under the name of the file
  - these results are themselves a map where the results of each query are stored under the query name
  - the individual fields in the result are the captured values in the treesitter query
- finally, the results are rendered using the configured template
  - the results are passed as the "Results" field
  - the command line flags are also passed to the result template