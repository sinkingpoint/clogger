package pipeline

import (
	"context"
	"sync"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
)

type assignedMessage struct {
	src string
	msg []clogger.Message
}

type Pipeline struct {
	KillChannel chan bool
	Inputs      map[string]inputs.Inputter
	Outputs     map[string]outputs.Outputter
	Pipes       map[string][]string
}

func NewPipeline(inputs map[string]inputs.Inputter, outputs map[string]outputs.Outputter, pipes map[string][]string) *Pipeline {
	return &Pipeline{
		Inputs:      inputs,
		Outputs:     outputs,
		Pipes:       pipes,
		KillChannel: make(chan bool),
	}
}

func (p *Pipeline) Run() sync.WaitGroup {
	wg := sync.WaitGroup{}
	inputPipes := make(map[string]chan []clogger.Message, len(p.Inputs))
	outputPipes := make(map[string]chan []clogger.Message, len(p.Outputs))
	outputKillPipes := make(map[string]chan bool, len(p.Outputs))
	inputAggKillPipes := make(map[string]chan bool, len(p.Inputs))

	aggInputChannel := make(chan assignedMessage)

	for name, input := range p.Inputs {
		inputPipes[name] = make(chan []clogger.Message)

		wg.Add(1)
		go func(input inputs.Inputter, inputChannel chan []clogger.Message) {
			defer wg.Done()
			input.Run(context.Background(), inputChannel)
		}(input, inputPipes[name])

		wg.Add(1)
		inputAggKillPipes[name] = make(chan bool)
		go func(name string, inputPipe chan []clogger.Message, kill chan bool) {
			defer wg.Done()
		outer:
			for {
				select {
				case <-kill:
					break outer
				case msg := <-inputPipe:
					aggInputChannel <- assignedMessage{
						name,
						msg,
					}
				}
			}

		}(name, inputPipes[name], inputAggKillPipes[name])
	}

	for name, output := range p.Outputs {
		outputPipes[name] = make(chan []clogger.Message)
		outputKillPipes[name] = make(chan bool)
		wg.Add(1)
		go func(output outputs.Outputter, pipe chan []clogger.Message, kill chan bool) {
			defer wg.Done()
			outputs.StartOutputter(pipe, output, kill)
		}(output, outputPipes[name], outputKillPipes[name])
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
	outer:
		for {
			select {
			case <-p.KillChannel:
				for _, input := range p.Inputs {
					input.Kill()
				}

				for _, c := range inputAggKillPipes {
					c <- true
				}

				for _, c := range outputKillPipes {
					c <- true
				}

				break outer
			case msg := <-aggInputChannel:
				if pipes, ok := p.Pipes[msg.src]; ok {
					for _, to := range pipes {
						outputPipes[to] <- msg.msg
					}
				}
			}
		}
	}()

	return wg
}
