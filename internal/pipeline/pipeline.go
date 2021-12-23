package pipeline

import (
	"context"
	"sync"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
)

type Pipeline struct {
	KillChannel chan bool
	Inputs      []inputs.Inputter
	Outputs     []outputs.Sender
}

func NewPipeline(inputs []inputs.Inputter, outputs []outputs.Sender) *Pipeline {
	return &Pipeline{
		Inputs:      inputs,
		Outputs:     outputs,
		KillChannel: make(chan bool),
	}
}

func (p *Pipeline) Run() sync.WaitGroup {
	inputChannel := make(chan []clogger.Message)
	wg := sync.WaitGroup{}
	for i := range p.Inputs {
		wg.Add(1)
		go func(input inputs.Inputter) {
			defer wg.Done()
			input.Run(context.Background(), inputChannel)
		}(p.Inputs[i])
	}

	outputChans := make([]chan []clogger.Message, len(p.Outputs))

	for i := range p.Outputs {
		wg.Add(1)
		outputChans[i] = make(chan []clogger.Message)
		go func(output outputs.Sender, flush chan []clogger.Message) {
			defer wg.Done()
			outputs.StartOutputter(flush, output)
		}(p.Outputs[i], outputChans[i])
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
	outer:
		for {
			select {
			case <-p.KillChannel:
				// TODO: Pass the Kill signal to all the running things
				break outer
			case message := <-inputChannel:
				for i := range outputChans {
					outputChans[i] <- message
				}
			}
		}
	}()

	return wg
}
