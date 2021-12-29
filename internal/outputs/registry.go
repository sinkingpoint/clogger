package outputs

import (
	"fmt"
)

type outputterConstructor = func(rawConf interface{}) (Outputter, error)
type configConstructor = func(map[string]string) (interface{}, error)

var outputsRegistry = newRegistry()

func init() {
	outputsRegistry.Register("stdout", func(rawConf map[string]string) (interface{}, error) {
		conf, err := NewSendConfigFromRaw(rawConf)
		if err != nil {
			return nil, err
		}

		return StdOutputterConfig{
			SendConfig: conf,
		}, nil
	}, func(rawConf interface{}) (Outputter, error) {
		conf, ok := rawConf.(StdOutputterConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config passed to StdOutputter")
		}

		return NewStdOutputter(conf)
	})
}

type outputterRegistry struct {
	constructorRegistry map[string]outputterConstructor
	configRegistry      map[string]configConstructor
}

func newRegistry() outputterRegistry {
	return outputterRegistry{
		constructorRegistry: make(map[string]outputterConstructor),
		configRegistry:      make(map[string]configConstructor),
	}
}

func (r *outputterRegistry) Register(name string, configGen configConstructor, constructor outputterConstructor) {
	r.constructorRegistry[name] = constructor
	r.configRegistry[name] = configGen
}

func HasConstructorFor(name string) bool {
	_, ok := outputsRegistry.configRegistry[name]
	return ok
}

func Construct(name string, config map[string]string) (Outputter, error) {
	if configMaker, ok := outputsRegistry.configRegistry[name]; ok {
		config, err := configMaker(config)
		if err != nil {
			return nil, err
		}

		if inputMaker, ok := outputsRegistry.constructorRegistry[name]; ok {
			return inputMaker(config)
		} else {
			return nil, fmt.Errorf("failed to find outputter `%s`", name)
		}
	} else {
		return nil, fmt.Errorf("failed to find outputter `%s`", name)
	}
}
