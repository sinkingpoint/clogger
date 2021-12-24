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
	killChannels := make([]chan bool, len(p.Outputs))

	for i := range p.Outputs {
		wg.Add(1)
		outputChans[i] = make(chan []clogger.Message)
		killChannels[i] = make(chan bool)

		go func(output outputs.Sender, flush chan []clogger.Message, kill chan bool) {
			defer wg.Done()
			outputs.StartOutputter(flush, output, kill)
		}(p.Outputs[i], outputChans[i], killChannels[i])
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
	outer:
		for {
			select {
			case <-p.KillChannel:
				// TODO: Pass the Kill signal to all the running things
				for i := range p.Outputs {
					killChannels[i] <- true
				}
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
