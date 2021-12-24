package outputs

import (
	"fmt"
)

type outputterConstructor = func(rawConf interface{}) (Outputter, error)
type configConstructor = func(map[string]string) (interface{}, error)

var outputsRegistry = NewRegistry()

func init() {
	outputsRegistry.Register("stdout", func(rawConf map[string]string) (interface{}, error) {
		conf, err := NewSendConfigFromRaw(rawConf)
		if err != nil {
			return nil, err
		}

		if format, ok := rawConf["format"]; ok {
			return StdOutputterConfig{
				SendConfig: conf,
				Formatter:  format,
			}, nil
		}

		return StdOutputterConfig{
			SendConfig: conf,
			Formatter:  "json",
		}, nil
	}, func(rawConf interface{}) (Outputter, error) {
		conf, ok := rawConf.(StdOutputterConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config passed to StdOutputter")
		}

		return NewStdOutputter(conf)
	})
}

type OutputterRegistry struct {
	constructorRegistry map[string]outputterConstructor
	configRegistry      map[string]configConstructor
}

func NewRegistry() OutputterRegistry {
	return OutputterRegistry{
		constructorRegistry: make(map[string]outputterConstructor),
		configRegistry:      make(map[string]configConstructor),
	}
}

func (r *OutputterRegistry) Register(name string, configGen configConstructor, constructor outputterConstructor) {
	r.constructorRegistry[name] = constructor
	r.configRegistry[name] = configGen
}

func Construct(name string, config map[string]string) (Outputter, error) {
	fmt.Println("Constructing ", name, " with config ", config)
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
