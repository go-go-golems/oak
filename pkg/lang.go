package pkg

import (
	"fmt"
	"github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/cue"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	//"github.com/smacker/go-tree-sitter/lua"
	"github.com/smacker/go-tree-sitter/ocaml"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/svelte"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
	"path"
)

func LanguageNameToSitterLanguage(name string) (*sitter.Language, error) {
	switch name {
	case "bash":
		return bash.GetLanguage(), nil
	case "c":
		return c.GetLanguage(), nil
	case "cpp":
		return cpp.GetLanguage(), nil
	case "csharp":
		return csharp.GetLanguage(), nil
	case "css":
		return css.GetLanguage(), nil
	case "cue":
		return cue.GetLanguage(), nil
	case "dockerfile":
		return dockerfile.GetLanguage(), nil
	case "elixir":
		return elixir.GetLanguage(), nil
	case "elm":
		return elm.GetLanguage(), nil
	case "golang":
		fallthrough
	case "go":
		return golang.GetLanguage(), nil
	case "hcl":
		return hcl.GetLanguage(), nil
	case "html":
		return html.GetLanguage(), nil
	case "java":
		return java.GetLanguage(), nil
	case "javascript":
		return javascript.GetLanguage(), nil
	case "kotlin":
		return kotlin.GetLanguage(), nil
	//case "lua":
	//	return lua.GetLanguage(), nil
	case "ocaml":
		return ocaml.GetLanguage(), nil
	case "php":
		return php.GetLanguage(), nil
	case "protobuf":
		return protobuf.GetLanguage(), nil
	case "python":
		return python.GetLanguage(), nil
	case "ruby":
		return ruby.GetLanguage(), nil
	case "rust":
		return rust.GetLanguage(), nil
	case "scala":
		return scala.GetLanguage(), nil
	case "svelte":
		return svelte.GetLanguage(), nil
	case "toml":
		return toml.GetLanguage(), nil
	case "typescript":
		return typescript.GetLanguage(), nil
	case "tsx":
		return tsx.GetLanguage(), nil
	case "yaml":
		return yaml.GetLanguage(), nil
	default:
		return nil, fmt.Errorf("unsupported language name: %s", name)
	}
}

var (
	fileEndingToLanguageName = map[string]string{
		"*.sh":       "bash",
		"*.c":        "c",
		"*.cpp":      "cpp",
		"*.h":        "cpp",
		"*.hpp":      "cpp",
		"*.cs":       "csharp",
		"*.css":      "css",
		"*.cue":      "cue",
		"Dockerfile": "dockerfile",
		"*.ex":       "elixir",
		"*.elm":      "elm",
		"*.go":       "go",
		"*.hcl":      "hcl",
		"*.tf":       "hcl",
		"*.html":     "html",
		"*.java":     "java",
		"*.js":       "javascript",
		"*.jsx":      "javascript",
		"*.kt":       "kotlin",
		//"*.lua":      "lua",
		"*.ml":     "ocaml",
		"*.mli":    "ocaml",
		"*.php":    "php",
		"*.proto":  "protobuf",
		"*.py":     "python",
		"*.rb":     "ruby",
		"*.rs":     "rust",
		"*.scala":  "scala",
		"*.svelte": "svelte",
		"*.toml":   "toml",
		"*.ts":     "typescript",
		"*.tsx":    "tsx",
		"*.yml":    "yaml",
		"*.yaml":   "yaml",
	}
)

func GetLanguageGlobs(lang string) ([]string, error) {
	ret := []string{}

	for filename, lang_ := range fileEndingToLanguageName {
		if lang_ == lang {
			ret = append(ret, fmt.Sprintf("**/%s", filename))
		}
	}

	if len(ret) == 0 {
		return nil, fmt.Errorf("unsupported language name: %s", lang)
	} else {
		return ret, nil
	}
}

func FileNameToSitterLanguage(filename string) (*sitter.Language, error) {
	baseName := path.Base(filename)
	for ending, name := range fileEndingToLanguageName {
		// use glob
		matched, err := path.Match(ending, baseName)
		if err != nil {
			return nil, err
		}
		if matched {
			return LanguageNameToSitterLanguage(name)
		}
	}
	return nil, fmt.Errorf("unsupported file name: %s", filename)
}
