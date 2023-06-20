---
Title: Output matches as structured data
Slug: glaze-output
Commands:
  - oak
  - glaze
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

## Output query results as structured data

To output query results as raw, structured data that can further be processed
by the middlewares provided by [glazed](https://github.com/go-go-golems/glazed),
use the `glaze` verb followed by the name of your command.

It will output the query results as an array of objects.

``` 
❯ oak glaze example1 test-inputs/test.go
+---------------------+----------------------+------------+----------------------------+--------------+
| file                | query                | capture    | type                       | text         |
+---------------------+----------------------+------------+----------------------------+--------------+
| test-inputs/test.go | functionDeclarations | name       | identifier                 | foo          |
| test-inputs/test.go | functionDeclarations | parameters | parameter_list             | (s string)   |
| test-inputs/test.go | functionDeclarations | name       | identifier                 | main         |
| test-inputs/test.go | functionDeclarations | parameters | parameter_list             | ()           |
| test-inputs/test.go | functionDeclarations | name       | identifier                 | someFunction |
| test-inputs/test.go | functionDeclarations | parameters | parameter_list             | ()           |
| test-inputs/test.go | functionDeclarations | name       | identifier                 | printString  |
| test-inputs/test.go | functionDeclarations | parameters | parameter_list             | (s string)   |
| test-inputs/test.go | importStatements     | path       | interpreted_string_literal | "fmt"        |
+---------------------+----------------------+------------+----------------------------+--------------+
```

You can then use all the familiar glaze flags to manipulate the output, for example outputting the results as JSON:

```
❯ oak glaze example1 test-inputs/test.go --output json --fields type,text,capture
[
{
  "capture": "parameters",
  "type": "parameter_list",
  "text": "(s string)"
},
{
  "capture": "name",
  "type": "identifier",
  "text": "foo"
},
...
```