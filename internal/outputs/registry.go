package outputs

import (
	"fmt"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type outputterConstructor = func(rawConf interface{}) (clogger.Sender, error)
type configConstructor = func(map[string]interface{}) (interface{}, error)

var OutputsRegistry = NewRegistry()

func init() {
	OutputsRegistry.Register("stdout", func(rawConf map[string]interface{}) (interface{}, error) {
		if format, ok := rawConf["format"]; ok {
			if s, ok := format.(string); ok {
				return StdOutputterConfig{
					Formatter: s,
				}, nil
			} else {
				return nil, fmt.Errorf("invalid formatter specified: `%s`", format)
			}
		}

		return StdOutputterConfig{
			Formatter: "json",
		}, nil
	}, func(rawConf interface{}) (clogger.Sender, error) {
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
