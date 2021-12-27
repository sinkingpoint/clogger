package pipeline

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
)

type assignedMessage struct {
	src string
	msg []clogger.Message
}

type Pipeline struct {
	Inputs      map[string]inputs.Inputter
	Outputs     map[string]outputs.Outputter
	Pipes       map[string][]string
	killChannel chan bool

	// we have a seperate wait group for inputs so that we can kill them seperatly
	wg sync.WaitGroup
}

func NewPipeline(inputs map[string]inputs.Inputter, outputs map[string]outputs.Outputter, pipes map[string][]string) *Pipeline {
	return &Pipeline{
		Inputs:      inputs,
		Outputs:     outputs,
		Pipes:       pipes,
		killChannel: make(chan bool, 1),
		wg:          sync.WaitGroup{},
	}
}

func (p *Pipeline) Kill() {
	p.killChannel <- true
	p.wg.Wait()
}

func (p *Pipeline) Wait() {
	p.wg.Wait()
}

func (p *Pipeline) Run() {
	inputPipes := make(map[string]chan []clogger.Message, len(p.Inputs))
	outputPipes := make(map[string]chan []clogger.Message, len(p.Outputs))

	aggInputChannel := make(chan assignedMessage, 10)

	inputWg := sync.WaitGroup{}
	inputAggWg := sync.WaitGroup{}
	outputWg := sync.WaitGroup{}

	for name, input := range p.Inputs {
		inputPipes[name] = make(chan []clogger.Message, 10)

		inputWg.Add(1)
		go func(input inputs.Inputter, inputChannel chan []clogger.Message) {
			defer inputWg.Done()
			defer close(inputChannel)
			err := input.Run(context.Background(), inputChannel)
			if err != nil {
				log.Err(err).Str("input_name", name).Msg("Failed to start inputter")
			}
		}(input, inputPipes[name])

		inputAggWg.Add(1)
		go func(name string, inputPipe chan []clogger.Message) {
			defer inputAggWg.Done()
		outer:
			for {
				msg, ok := <-inputPipe
				if !ok {
					break outer
				}
				aggInputChannel <- assignedMessage{
					name,
					msg,
				}
			}
		}(name, inputPipes[name])
	}

	for name, output := range p.Outputs {
		outputPipes[name] = make(chan []clogger.Message, 10)
		outputWg.Add(1)
		go func(output outputs.Outputter, pipe chan []clogger.Message) {
			defer func() {
				outputWg.Done()
			}()
			outputs.StartOutputter(pipe, output)
		}(output, outputPipes[name])
	}

	p.wg.Add(1)
	go func() {
		for {
			msg, ok := <-aggInputChannel
			if pipes, ok := p.Pipes[msg.src]; ok {
				for _, to := range pipes {
					outputPipes[to] <- msg.msg
				}
			}

			if !ok {
				break
			}
		}

		for _, pipe := range outputPipes {
			close(pipe)
		}
	}()

	// A note on ordering here (UPDATE THIS IF YOU CHANGE ANYTHING BELOW THIS LINE):
	// 1. We kill the inputs so we stop enqueuing new messages, and then wait for all inputs to exit
	// 2. The closing of the input channels kills the aggregator channels that read from the inputs, to flush all messages to the outputs
	// 3. Once all the aggregator channels are closed, we close the firehose channel, to flush all the messages to the outputs
	// 4. The firehose channel being closed closes the pipeline channel that reads from the firehose
	// 5. In closing, the pipeline channel closes all the output channels
	// 6. The output channels being closed forces the outputs to flush and exit
	go func() {
		<-p.killChannel
		for _, input := range p.Inputs {
			input.Kill()
		}
		inputWg.Wait()

		inputAggWg.Wait()
		close(aggInputChannel)

		outputWg.Wait()
		p.wg.Done()
	}()
}
