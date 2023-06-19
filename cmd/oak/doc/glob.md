---
Title: Parsing multiple files recursively
Slug: glob
Topics:
  - oak
Commands:
  - oak
Flags:
  - recurse
  - glob
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

## Parsing multiple files

Often, it is useful to run an oak command against a directory recursively, or even against a full repository.
You can use the `--recurse` and `--glob` files to do so.

`--recurse` will find all the files whose ending matches the default line endings for the configured language of the
command. For example `--recurse` on a go command will find all files ending in `.go`.

``` 
❯ oak example1 --recurse test-inputs 
File: test-inputs/test.go

Function Declarations:
- foo(s string) 
- main() 
- someFunction() 
- printString(s string) 

Import Statements:
- path: "fmt"

```

`--glob` will find all files whose name matches the provided glob patterns (multiple patterns can be provided
either by passing the flag multiple times, or providing a comma-separated list). 

```
❯ oak example1 --glob 'test-inputs/*.go,**/pkg/queries.go' .
File: pkg/queries.go

Function Declarations:
...

File: test-inputs/test.go

Function Declarations:
...


```

You can still pass filenames explicitly amongst the list of directories that will be searched recursively:

``` 
❯ go run ./cmd/oak example1 --glob **/queries.go . ./test-inputs/test.go
File: ./test-inputs/test.go

Function Declarations:
...

File: pkg/queries.go

Function Declarations:
...
```