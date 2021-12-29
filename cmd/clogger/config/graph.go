package config

import (
	"fmt"
	"strings"
)

type node struct {
	name  string
	attrs map[string]string
}

type edge struct {
	from  string
	to    string
	attrs map[string]string
}

type ConfigGraph struct {
	name       string
	attrs      map[string]string
	subGraphs  map[string]ConfigGraph
	nodes      map[string]node
	connectors []edge
}

func newConfigGraph() ConfigGraph {
	return ConfigGraph{
		name:       "",
		attrs:      make(map[string]string),
		subGraphs:  make(map[string]ConfigGraph),
		nodes:      make(map[string]node),
		connectors: make([]edge, 0, 10),
	}
}

func (c *ConfigGraph) SetStrict(strict bool) error {
	return nil
}

func (c *ConfigGraph) SetDir(directed bool) error {
	return nil
}

func (c *ConfigGraph) SetName(name string) error {
	c.name = name
	return nil
}

func (c *ConfigGraph) AddPortEdge(src, srcPort, dst, dstPort string, directed bool, attrs map[string]string) error {
	return c.AddEdge(src, dst, directed, attrs)
}

func (c *ConfigGraph) AddEdge(src, dst string, directed bool, attrs map[string]string) error {
	if !directed {
		return fmt.Errorf("edges in the Config Graph must be directed")
	}

	c.connectors = append(c.connectors, edge{
		from:  src,
		to:    dst,
		attrs: attrs,
	})

	return nil
}

func (c *ConfigGraph) AddNode(parentGraph string, name string, attrs map[string]string) error {
	if parentGraph == c.name {
		if _, ok := c.nodes[name]; ok {
			return fmt.Errorf("config graph already contains a node called `%s`", name)
		}

		for i := range attrs {
			attrs[i] = strings.Trim(attrs[i], "\"")
		}

		c.nodes[name] = node{
			name,
			attrs,
		}
	} else {
		// Node must be in a subgraph
		// NOTE: This only supports one level of nesting
		if sub, ok := c.subGraphs[parentGraph]; ok {
			sub.AddNode(parentGraph, name, attrs)
		} else {
			return fmt.Errorf("failed to find subgraph `%s` to add node", parentGraph)
		}
	}

	return nil
}

func (c *ConfigGraph) AddAttr(parentGraph string, field, value string) error {
	if parentGraph == c.name {
		if _, ok := c.attrs[field]; ok {
			return fmt.Errorf("graph already has an attribute `%s`", field)
		}

		value := strings.Trim(value, "\"")

		c.attrs[field] = value
	} else {
		if sub, ok := c.subGraphs[parentGraph]; ok {
			sub.AddAttr(parentGraph, field, value)
		} else {
			return fmt.Errorf("failed to find subgraph `%s` to add node", parentGraph)
		}
	}
	return nil
}

func (c *ConfigGraph) AddSubGraph(parentGraph string, name string, attrs map[string]string) error {
	if parentGraph == c.name {
		if _, ok := c.attrs[name]; ok {
			return fmt.Errorf("graph already has an subgraph `%s`", name)
		}

		graph := newConfigGraph()
		graph.name = name
		graph.attrs = attrs

		c.subGraphs[name] = graph
	} else {
		// We only support one level of nesting for now, error if we're trying to add a subgraph to a subgraph
		return fmt.Errorf("clogger only supports one level of subgraph nesting in configs")
	}
	return nil
}

func (c *ConfigGraph) String() string {
	return ""
}
