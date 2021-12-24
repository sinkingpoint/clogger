package inputs

import (
	"fmt"
	"strings"
)

type inputterConstructor = func(rawConf interface{}) (Inputter, error)
type configConstructor = func(rawConf map[string]string) (interface{}, error)

var inputsRegistry = NewRegistry()

func init() {
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

type InputterRegistry struct {
	constructorRegistry map[string]inputterConstructor
	configRegistry      map[string]configConstructor
}

func NewRegistry() InputterRegistry {
	return InputterRegistry{
		constructorRegistry: make(map[string]inputterConstructor),
		configRegistry:      make(map[string]configConstructor),
	}
}

func (r *InputterRegistry) Register(name string, configGen configConstructor, constructor inputterConstructor) {
	r.constructorRegistry[name] = constructor
	r.configRegistry[name] = configGen
}

func Construct(name string, config map[string]string) (Inputter, error) {
	name = strings.ToLower(name)
	if configMaker, ok := inputsRegistry.configRegistry[name]; ok {
		config, err := configMaker(config)
		if err != nil {
			return nil, err
		}

		if inputMaker, ok := inputsRegistry.constructorRegistry[name]; ok {
			return inputMaker(config)
		} else {
			return nil, fmt.Errorf("failed to find inputter `%s`", name)
		}
	} else {
		return nil, fmt.Errorf("failed to find inputter `%s`", name)
	}
}
