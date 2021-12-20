package inputs

import "fmt"

type inputterConstructor = func(rawConf interface{}) (Inputter, error)
type configConstructor = func(map[string]interface{}) (interface{}, error)

var InputsRegistry = NewRegistry()

func init() {
	InputsRegistry.Register("journald", func(rawConf map[string]interface{}) (interface{}, error) {
		return JournalDInputConfig{
			BatchSize: 10,
		}, nil
	}, func(conf interface{}) (Inputter, error) {
		if c, ok := conf.(*JournalDInputConfig); ok {
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
