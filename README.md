# GO GO OAK

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

Oak allows the user to provide [tree-sitter](https://tree-sitter.github.io/tree-sitter/) queries
in a YAML file and use the resulting captures to expand a go template.

## Background

When prompting LLMs for programming, it is very useful to provide some context about
the code that you want to generate, for example out of your current codebase.

Just copy pasting code gets you really far, but it eats a lot of tokens, and often 
confuses the LLM. Minimal prompts are often much more effective (see [Exploring coding with LLMs](https://share.descript.com/view/CDetEUb5doZ)).
In order to quickly generate minimal prompts out of an existing codebase, we can use `oak`,
which follows the pattern of tools like [glaze](https://github.com/go-go-golems/glazed),
[sqleton](https://github.com/go-go-golems/sqleton) or [pinocchio](https://github.com/go-go-golems/geppetto)
which allow the user to declare "commands" in a YAML file.

## Getting started

After installing oak by downloading the proper release binary, you can start creating oak commands
by storing them in your command repository (a directory containing all your oak queries).

To get started, use `oak help create-query` to get an introduction to creating a new verb.

You can also run [example1](./cmd/oak/queries/example1.yaml) against [a go file](./test-inputs/test.go)
to get a list of imports and function declarations.

```
❯ oak example1 ./test-inputs/test.go
File: ./test-inputs/test.go

Function Declarations:
- foo(s string) 
- main() 
- someFunction() 
- printString(s string) 

Import Statements:
- path: "fmt"
```

Commands can be run against whole directories or with explicit globs. For more information, use `oak help glob`.

``` 
❯ oak example1 --glob **/queries.go . ./test-inputs/test.go
File: ./test-inputs/test.go

Function Declarations:
...

File: pkg/queries.go

Function Declarations:
...
```

## Rendering the query templates

Queries are themselves go templates that will get expanded based on the command-line flags
defined in the oak YAML file.

You can print the queries to get a preview of what they would look like rendered.

For example, for the following YAML:

```yaml
name: equals
short: Find expressions where the right side equals a number.
flags:
  - name: number
    type: int
    help: The number to compare against
    default: 1
language: go

queries:
  - name: testPredicate
    query: |
      (binary_expression
         left: (_) @left
         right: (_) @right
       (#eq? @right {{ .number }}))

template: |
  {{ range .testPredicate.Matches }}
  - {{ .left.Text }} - {{.right.Text}}{{ end }}

```

We can either use the default value: 

``` 
❯ oak equals --print-queries
- name: testPredicate
  query: |
    (binary_expression
       left: (_) @left
       right: (_) @right
     (#eq? @right 1))
```

or provide a custom number:

```
❯ oak equals --print-queries --number 23 
- name: testPredicate
  query: |
    (binary_expression
       left: (_) @left
       right: (_) @right
     (#eq? @right 23))
```
