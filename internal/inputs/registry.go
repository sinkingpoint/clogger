package inputs

import (
	"fmt"
	"strings"
)

// configConstructor takes the raw config map and constructs a relevant structured config
// that will be used to construct an Inputter in an inputterConstructor
type configConstructor = func(rawConf map[string]string) (interface{}, error)

// inputterConstructor is a function that takes a config (outputted from a configConstructor)
// and returns an inputter generated from that config
type inputterConstructor = func(rawConf interface{}) (Inputter, error)

var inputsRegistry = newRegistry()

func init() {
	// JournalDInput that reads data from the journald stream
	inputsRegistry.Register("journald", func(rawConf map[string]string) (interface{}, error) {
		conf := NewRecvConfig()

		return conf, nil
	}, func(conf interface{}) (Inputter, error) {
		if c, ok := conf.(RecvConfig); ok {
			return NewJournalDInput(c)
		}

		return nil, fmt.Errorf("invalid config passed to journald input")
	})
}

// inputterRegistry is a registry of functions that can be used to construct Inputters
// from string configs
type inputterRegistry struct {
	constructorRegistry map[string]inputterConstructor
	configRegistry      map[string]configConstructor
}

func newRegistry() inputterRegistry {
	return inputterRegistry{
		constructorRegistry: make(map[string]inputterConstructor),
		configRegistry:      make(map[string]configConstructor),
	}
}

// Register is a convenience method that registers the given constructors against the name
// so that we can construct things with those constructors
func (r *inputterRegistry) Register(name string, configGen configConstructor, constructor inputterConstructor) {
	r.constructorRegistry[name] = constructor
	r.configRegistry[name] = configGen
}

// Construct constructs an Inputter from the given name and config map,
// returning an error if the name or config is invalid
func Construct(name string, config map[string]string) (Inputter, error) {
	name = strings.ToLower(name)
	if configMaker, ok := inputsRegistry.configRegistry[name]; ok {
		// Construction is in two steps, because this makes the code a bit cleaner

		// First, parse the unstructured config into a proper struct
		// This allows us to do all the validation up front and not in the constructor of the actual object
		config, err := configMaker(config)
		if err != nil {
			return nil, err
		}

		// Second, use that config struct to construct the actual inputter
		if inputMaker, ok := inputsRegistry.constructorRegistry[name]; ok {
			return inputMaker(config)
		} else {
			return nil, fmt.Errorf("failed to find inputter `%s`", name)
		}
	} else {
		return nil, fmt.Errorf("failed to find inputter `%s`", name)
	}
}
