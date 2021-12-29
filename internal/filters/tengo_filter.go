package filters

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/d5/tengo/v2"
	"github.com/sinkingpoint/clogger/internal/clogger"
)

type TengoFilterConfig struct {
	Path string
}

type TengoFilter struct {
	compiled *tengo.Compiled
	failOpen bool
}

func NewTengoFilterFromString(s []byte) (*TengoFilter, error) {
	script := tengo.NewScript(s)
	script.Add("message", nil)

	compiled, err := script.Compile()
	if err != nil {
		return nil, err
	}

	return &TengoFilter{
		compiled: compiled,
	}, nil
}

func NewTengoFilterFromConf(conf TengoFilterConfig) (*TengoFilter, error) {
	bytes, err := ioutil.ReadFile(conf.Path)
	if err != nil {
		return nil, err
	}

	return NewTengoFilterFromString(bytes)
}

func (t *TengoFilter) Filter(ctx context.Context, msg *clogger.Message) (shouldDrop bool, err error) {
	if err := t.compiled.Set("message", msg.ParsedFields); err != nil {
		return t.failOpen, err
	}

	if err := t.compiled.RunContext(ctx); err != nil {
		return t.failOpen, err
	}

	if message := t.compiled.Get("message").Map(); message != nil {
		msg.ParsedFields = message
	}

	shouldDrop = !t.failOpen
	if t.compiled.IsDefined("shouldDrop") {
		shouldDrop = t.compiled.Get("shouldDrop").Bool()
	}
	err = t.compiled.Get("err").Error()

	return shouldDrop, err
}

func init() {
	filtersRegistry.Register("tengo", func(rawConf map[string]string) (interface{}, error) {
		if path, ok := rawConf["file"]; ok {
			return TengoFilterConfig{
				Path: path,
			}, nil
		} else {
			return nil, fmt.Errorf("missing required configuration `file` for Tengo filter")
		}
	}, func(rawConf interface{}) (Filter, error) {
		if conf, ok := rawConf.(TengoFilterConfig); ok {
			return NewTengoFilterFromConf(conf)
		} else {
			return nil, fmt.Errorf("BUG: invalid type for Tengo filter configuration (expected TengoFilterConfig)")
		}
	})
}
