package config

import (
	"fmt"
	"io/ioutil"

	"github.com/awalterschulze/gographviz"
	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/filters"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
	"github.com/sinkingpoint/clogger/internal/pipeline"
)

func LoadConfigFile(path string) (*pipeline.Pipeline, error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	graphAst, err := gographviz.ParseString(string(body))
	if err != nil {
		return nil, err
	}

	configGraph := newConfigGraph()
	if err := gographviz.Analyse(graphAst, &configGraph); err != nil {
		return nil, err
	}

	return configGraph.ToPipeline()
}

func (c *ConfigGraph) ToPipeline() (*pipeline.Pipeline, error) {
	if len(c.edges) == 0 {
		log.Warn().Msg("No connectors in this pipeline. It wont do anything")
	}

	inputsMemoize := make(map[string]inputs.Inputter)
	outputsMemoize := make(map[string]outputs.Outputter)
	filtersMemoize := make(map[string]filters.Filter)
	pipes := make(map[string][]pipeline.Link)
	for i := range c.edges {
		edge := c.edges[i]

		_, hasInput := inputsMemoize[edge.from]
		_, hasFilter := filtersMemoize[edge.from]

		// TODO @sinkingpoint: This is full of duplication and should be refactored
		ty := edge.attrs["type"]
		if ty == "Buffer" {
			if _, ok := outputsMemoize[edge.from]; !ok {
				fromData := c.nodes[edge.from]
				if ty, ok := fromData.attrs["type"]; ok {
					if outputs.HasConstructorFor(ty) {
						to, err := outputs.Construct(ty, fromData.attrs)
						if err != nil {
							return nil, err
						}

						outputsMemoize[fromData.name] = to
					} else {
						return nil, fmt.Errorf("no output type called `%s`", ty)
					}
				} else {
					return nil, fmt.Errorf("node `%s` is missing a `type` attribute", edge.from)
				}
			}

			toData := c.nodes[edge.to]
			if ty, ok := toData.attrs["type"]; ok {
				if outputs.HasConstructorFor(ty) {
					to, err := outputs.Construct(ty, toData.attrs)
					if err != nil {
						return nil, err
					}

					outputsMemoize[toData.name] = to
				} else {
					return nil, fmt.Errorf("no output type called `%s`", ty)
				}
			} else {
				return nil, fmt.Errorf("node `%s` is missing a `type` attribute", edge.from)
			}

			pipes[edge.from] = append(pipes[edge.from], pipeline.Link{
				To:   edge.to,
				Type: pipeline.LINK_TYPE_BUFFER,
			})
		} else {
			if !hasFilter && !hasInput {
				fromData := c.nodes[edge.from]
				if ty, ok := fromData.attrs["type"]; ok {
					if inputs.HasConstructorFor(ty) {
						from, err := inputs.Construct(ty, fromData.attrs)
						if err != nil {
							return nil, err
						}

						inputsMemoize[fromData.name] = from
					} else if filters.HasConstructorFor(ty) {
						from, err := filters.Construct(ty, fromData.attrs)
						if err != nil {
							return nil, err
						}

						filtersMemoize[fromData.name] = from
					} else {
						return nil, fmt.Errorf("no such type type `%s`", ty)
					}
				} else {
					return nil, fmt.Errorf("node `%s` is missing a `type` attribute", edge.from)
				}
			}

			if _, ok := outputsMemoize[edge.to]; !ok {
				toData := c.nodes[edge.to]
				if ty, ok := toData.attrs["type"]; ok {
					if outputs.HasConstructorFor(ty) {
						to, err := outputs.Construct(ty, toData.attrs)
						if err != nil {
							return nil, err
						}

						outputsMemoize[toData.name] = to
					} else if filters.HasConstructorFor(ty) {
						to, err := filters.Construct(ty, toData.attrs)
						if err != nil {
							return nil, err
						}

						filtersMemoize[toData.name] = to
					} else {
						return nil, fmt.Errorf("no output or filter type called `%s`", ty)
					}
				} else {
					return nil, fmt.Errorf("node `%s` is missing a `type` attribute", edge.from)
				}
			}

			pipes[edge.from] = append(pipes[edge.from], pipeline.Link{
				To:   edge.to,
				Type: pipeline.LINK_TYPE_NORMAL,
			})
		}
	}

	return pipeline.NewPipeline(inputsMemoize, outputsMemoize, filtersMemoize, pipes), nil
}
