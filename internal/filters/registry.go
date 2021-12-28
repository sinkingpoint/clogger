package filters

import (
	"fmt"
	"strings"
)

// configConstructor takes the raw config map and constructs a relevant structured config
// that will be used to construct a Filter in an filterConstructor
type configConstructor = func(rawConf map[string]string) (interface{}, error)

// filterConstructor is a function that takes a config (outputted from a configConstructor)
// and returns a filter generated from that config
type filterConstructor = func(rawConf interface{}) (Filter, error)

var filtersRegistry = newRegistry()

// filterRegistry is a registry of functions that can be used to construct Filters
// from string configs
type filterRegistry struct {
	constructorRegistry map[string]filterConstructor
	configRegistry      map[string]configConstructor
}

func newRegistry() filterRegistry {
	return filterRegistry{
		constructorRegistry: make(map[string]filterConstructor),
		configRegistry:      make(map[string]configConstructor),
	}
}

// Register is a convenience method that registers the given constructors against the name
// so that we can construct things with those constructors
func (r *filterRegistry) Register(name string, configGen configConstructor, constructor filterConstructor) {
	r.constructorRegistry[name] = constructor
	r.configRegistry[name] = configGen
}

// Construct constructs a Filter from the given name and config map,
// returning an error if the name or config is invalid
func Construct(name string, config map[string]string) (Filter, error) {
	name = strings.ToLower(name)
	if configMaker, ok := filtersRegistry.configRegistry[name]; ok {
		// Construction is in two steps, because this makes the code a bit cleaner

		// First, parse the unstructured config into a proper struct
		// This allows us to do all the validation up front and not in the constructor of the actual object
		config, err := configMaker(config)
		if err != nil {
			return nil, err
		}

		// Second, use that config struct to construct the actual filter
		if filterMaker, ok := filtersRegistry.constructorRegistry[name]; ok {
			return filterMaker(config)
		} else {
			return nil, fmt.Errorf("failed to find filter `%s`", name)
		}
	} else {
		return nil, fmt.Errorf("failed to find filter `%s`", name)
	}
}
