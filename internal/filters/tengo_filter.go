package filters

import (
	"context"
	"io/ioutil"

	"github.com/d5/tengo/v2"
	"github.com/sinkingpoint/clogger/internal/clogger"
)

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

func NewTengoFilterFromFile(path string) (*TengoFilter, error) {
	bytes, err := ioutil.ReadFile(path)
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
