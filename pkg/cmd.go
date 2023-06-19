package pkg

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"gopkg.in/yaml.v3"
	"io"
)

//go:embed "layers/oak.yaml"
var oakLayerYaml string

type OakParameterLayer struct {
	layers.ParameterLayerImpl
}

func NewOakParameterLayer(
	options ...layers.ParameterLayerOptions,
) (*OakParameterLayer, error) {
	layer, err := layers.NewParameterLayerFromYAML([]byte(oakLayerYaml), options...)
	if err != nil {
		return nil, err
	}
	return &OakParameterLayer{
		ParameterLayerImpl: *layer,
	}, nil
}

type SitterQuery struct {
	// Name of the resulting variable after parsing
	Name string
	// Query contains the tree-sitter query that will be applied to the source code
	Query string
}

type OakCommandDescription struct {
	Language string        `yaml:"language,omitempty"`
	Queries  []SitterQuery `yaml:"queries"`
	Template string        `yaml:"template,omitempty"`

	Name   string                            `yaml:"name"`
	Short  string                            `yaml:"short"`
	Long   string                            `yaml:"long,omitempty"`
	Layout []*layout.Section                 `yaml:"layout,omitempty"`
	Flags  []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
	Layers []layers.ParameterLayer           `yaml:"layers,omitempty"`

	Parents []string `yaml:",omitempty"`
	Source  string   `yaml:",omitempty"`
}

type OakCommandLoader struct {
}

func (o *OakCommandLoader) LoadCommandFromYAML(
	s io.Reader,
	options ...cmds.CommandDescriptionOption,
) ([]cmds.Command, error) {
	ocd := &OakCommandDescription{}
	err := yaml.NewDecoder(s).Decode(ocd)
	if err != nil {
		return nil, err
	}

	oakLayer, err := NewOakParameterLayer()
	if err != nil {
		return nil, err
	}

	layers := append(ocd.Layers, oakLayer)

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithName(ocd.Name),
		cmds.WithShort(ocd.Short),
		cmds.WithLong(ocd.Long),
		cmds.WithFlags(ocd.Flags...),
		cmds.WithLayers(layers...),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"sources",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Files (or directories if recursing) to parse"),
				parameters.WithRequired(false),
			),
		),
		cmds.WithLayout(&layout.Layout{
			Sections: ocd.Layout,
		}),
	}
	options_ = append(options_, options...)

	oakCommand := NewOakCommand(
		cmds.NewCommandDescription(ocd.Name, options_...),
		WithQueries(ocd.Queries...),
		WithTemplate(ocd.Template),
		WithLanguage(ocd.Language),
	)

	return []cmds.Command{oakCommand}, nil
}

func (o *OakCommandLoader) LoadCommandAliasFromYAML(
	s io.Reader,
	options ...alias.Option,
) ([]*alias.CommandAlias, error) {
	return loaders.LoadCommandAliasFromYAML(s, options...)
}
