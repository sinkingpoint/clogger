package outputs

import "fmt"

type outputterConstructor = func(rawConf interface{}) (Outputter, error)
type configConstructor = func(map[string]interface{}) (interface{}, error)

var OutputsRegistry = NewRegistry()

func init() {
	OutputsRegistry.Register("stdout", func(rawConf map[string]interface{}) (interface{}, error) {
		if format, ok := rawConf["format"]; ok {
			if s, ok := format.(string); ok {
				return StdOutputterConfig{
					formatter: s,
				}, nil
			} else {
				return nil, fmt.Errorf("invalid formatter specified: `%s`", format)
			}
		}

		return StdOutputterConfig{
			formatter: "json",
		}, nil
	}, func(rawConf interface{}) (Outputter, error) {
		conf, ok := rawConf.(*StdOutputterConfig)
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
