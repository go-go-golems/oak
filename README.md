# GO GO OAK

```
 _______  _______    _______  _______    _______  _______  ___   _ 
|       ||       |  |       ||       |  |       ||   _   ||   | | |
|    ___||   _   |  |    ___||   _   |  |   _   ||  |_|  ||   |_| |
|   | __ |  | |  |  |   | __ |  | |  |  |  | |  ||       ||      _|
|   ||  ||  |_|  |  |   ||  ||  |_|  |  |  |_|  ||       ||     |_ 
|   |_| ||       |  |   |_| ||       |  |       ||   _   ||    _  |
|_______||_______|  |_______||_______|  |_______||__| |__||___| |_|
```

![image](https://user-images.githubusercontent.com/128441/233886270-13d0cdd7-ca86-4bea-982a-ffee978b2dd7.png)


---

Use tree-sitter to run queries against programming language files.

```
   ____   ___     ____   ___     ____   ___   _      _____  __  __  ____  
 / ___| / _ \   / ___| / _ \   / ___| / _ \ | |    | ____||  \/  |/ ___| 
| |  _ | | | | | |  _ | | | | | |  _ | | | || |    |  _|  | |\/| |\___ \ 
| |_| || |_| | | |_| || |_| | | |_| || |_| || |___ | |___ | |  | | ___) |
 \____| \___/   \____| \___/   \____| \___/ |_____||_____||_|  |_||____/ 
                                                                         
 _   _  ____   _____    ___     _     _  __  _____  ___  
| | | |/ ___| | ____|  / _ \   / \   | |/ / |_   _|/ _ \ 
| | | |\___ \ |  _|   | | | | / _ \  | ' /    | | | | | |
| |_| | ___) || |___  | |_| |/ ___ \ | . \    | | | |_| |
 \___/ |____/ |_____|  \___//_/   \_\|_|\_\   |_|  \___/ 
                                                         
 ____   ____   ___  _   _   ____    ___   ____   ____   _____  ____  
| __ ) |  _ \ |_ _|| \ | | / ___|  / _ \ |  _ \ |  _ \ | ____||  _ \ 
|  _ \ | |_) | | | |  \| || |  _  | | | || |_) || | | ||  _|  | |_) |
| |_) ||  _ <  | | | |\  || |_| | | |_| ||  _ < | |_| || |___ |  _ < 
|____/ |_| \_\|___||_| \_| \____|  \___/ |_| \_\|____/ |_____||_| \_\
                                                                     
 _____  ___     ____  _   _     _     ___   ____    
|_   _|/ _ \   / ___|| | | |   / \   / _ \ / ___|   
  | | | | | | | |    | |_| |  / _ \ | | | |\___ \   
  | | | |_| | | |___ |  _  | / ___ \| |_| | ___) |_ 
  |_|  \___/   \____||_| |_|/_/   \_\\___/ |____/(_)
                                                    
```

## Overview

When prompting LLMs for programming, it is very useful to provide some context about
the code that you want to generate, for example out of your current codebase.

Just copy pasting code gets you really far, but it eats a lot of tokens, and often 
confuses the LLM. Minimal prompts are often much more effective (see [Exploring coding with LLMs](https://share.descript.com/view/CDetEUb5doZ)).
In order to quickly generate minimal prompts out of an existing codebase, we can use `oak`,
which follows the pattern of tools like [glaze](https://github.com/go-go-golems/glazed),
[sqleton](https://github.com/go-go-golems/sqleton) or [pinocchio](https://github.com/go-go-golems/geppetto)
which allow the user to declare "commands" in a YAML file.

Oak allows the user to provide [tree-sitter](https://tree-sitter.github.io/tree-sitter/) queries
in a YAML file and use the resulting captures to expand a go template.

For example:

```yaml
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
  Function Declarations:
  {{ range .functionDeclarations.Matches }}
  - {{ .name.Text }}{{ .parameters.Text }}{{ end }}

  Import Statements:
  {{ range .importStatements.Matches }}
  - {{ if .name }}name: {{ .name.Text }}, {{end -}} path: {{ .path.Text }}{{ end }}
  
  Results:{{ range $v := .Results }}
    {{ $v.Name }}: {{ range $match := $v.Matches }}{{ range $captureName, $captureValue := $match }}
       {{ $captureName }}: {{ $captureValue.Text }}{{ end }}
    {{end}}{{ end }}
```

can be run against [test.go](test-inputs/test.go):

```
❯ go run ./cmd/oak run --query-file queries/example1.yaml --input-file test-inputs/test.go
Function Declarations:

- foo(s string)
- main()
- someFunction()
- printString(s string)

Import Statements:

- path: "fmt"

Results:
  functionDeclarations: 
     name: foo
     parameters: (s string)
  
     name: main
     parameters: ()
  
     name: someFunction
     parameters: ()
  
     name: printString
     parameters: (s string)
  
  importStatements: 
     path: "fmt"
```

## Brainstorming (2023-04-22)

Today I want to:

- [x] run queries against a file to understand how queries work

- [x] figure out how to filter query results to only return interesting
- [x] query against a terraform
- [] list supported languages

- [x] file pull request for the predicate fix (see https://github.com/smacker/go-tree-sitter/pull/102).

## Turn OakCommand into a glazed.Command

We have multiple ways of doing it:
- output templated structured data
- output a template
- output a straight list of capture data

Furthermore, we should make the queries templates that are expanded based on the input flags.
Also, exposing it as a verb means that we have to somehow give the whole thing the input files (as list?).

We can add those input file and language flags as a separate layer (also, --template, potentially multiple output files
for multiple input files).

I'm not sure how to do the structured templated output.

## Misc improvements

- add SQL grammar for tree-sitter, as well as anything else we might need
  - see https://github.com/smacker/go-tree-sitter/pull/58/files and https://github.com/smacker/go-tree-sitter/issues/57
