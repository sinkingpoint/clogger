package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/filters"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
)

type Pipeline struct {
	Inputs      map[string]inputs.Inputter
	Filters     map[string]filters.Filter
	Outputs     map[string]outputs.Outputter
	Pipes       map[string][]string
	RevPipes    map[string][]string
	killChannel chan bool
	debug       bool

	channels   map[string]clogger.MessageChannel
	closedLock sync.Mutex
	closed     map[string]bool

	wg sync.WaitGroup
}

func NewPipeline(inputs map[string]inputs.Inputter, outputs map[string]outputs.Outputter, filters map[string]filters.Filter, pipes map[string][]string) *Pipeline {
	revPipes := make(map[string][]string, len(pipes))

	for from, tos := range pipes {
		for _, to := range tos {
			revPipes[to] = append(revPipes[to], from)
		}
	}

	return &Pipeline{
		Inputs:      inputs,
		Outputs:     outputs,
		Filters:     filters,
		Pipes:       pipes,
		RevPipes:    revPipes,
		debug:       true,
		killChannel: make(chan bool, 1),
		closed:      make(map[string]bool, len(inputs)+len(outputs)+len(filters)),
		closedLock:  sync.Mutex{},
		channels:    make(map[string]clogger.MessageChannel, len(inputs)+len(filters)+len(outputs)),
		wg:          sync.WaitGroup{},
	}
}

func (p *Pipeline) handleClose(chanName string) {
	toHandle := []string{chanName}
	p.closedLock.Lock()
	defer p.closedLock.Unlock()
	p.closed[chanName] = true

	for len(toHandle) > 0 {
		chanName = toHandle[len(toHandle)-1]
		toHandle = toHandle[:len(toHandle)-1]
	outer:
		for _, dest := range p.Pipes[chanName] {
			if _, closed := p.closed[dest]; closed {
				continue
			}

			for _, src := range p.RevPipes[dest] {
				if _, closed := p.closed[src]; !closed {
					continue outer
				}
			}

			close(p.channels[dest])
			toHandle = append(toHandle, dest)
		}
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
	inputPipes := make(map[string]clogger.MessageChannel, len(p.Inputs))

	inputWg := sync.WaitGroup{}
	filterWg := sync.WaitGroup{}
	inputAggWg := sync.WaitGroup{}

	for name, input := range p.Inputs {
		inputPipes[name] = make(clogger.MessageChannel, 10)

		inputWg.Add(1)
		go func(input inputs.Inputter, inputChannel clogger.MessageChannel) {
			defer inputWg.Done()
			defer close(inputChannel)
			err := input.Run(context.Background(), inputChannel)
			if err != nil {
				log.Err(err).Str("input_name", name).Msg("Failed to start inputter")
			}
		}(input, inputPipes[name])

		inputAggWg.Add(1)
		go func(name string, inputPipe clogger.MessageChannel) {
			defer inputAggWg.Done()
			for msg := range inputPipe {
				for _, to := range p.Pipes[name] {
					if pipe, ok := p.channels[to]; ok {
						pipe <- msg
					} else {
						fmt.Printf("No destination found for `%s`\n", to)
					}
				}
			}

			p.handleClose(name)
		}(name, inputPipes[name])
	}

	for name, filter := range p.Filters {
		p.channels[name] = make(clogger.MessageChannel, 10)
		filterWg.Add(1)
		go func(name string, filter filters.Filter, inputPipe clogger.MessageChannel) {
			defer filterWg.Done()
			for msg := range inputPipe {
				output := make([]clogger.Message, 0, len(msg))

				for i := range msg {
					shouldDrop, err := filter.Filter(context.Background(), &msg[i])
					if err != nil {
						log.Warn().Err(err).Msg("Filter failed")
					}

					if !shouldDrop {
						output = append(output, msg[i])
					}
				}

				for _, to := range p.Pipes[name] {
					if pipe, ok := p.channels[to]; ok {
						pipe <- output
					} else {
						fmt.Printf("No destination found for `%s`\n", to)
					}
				}
			}
			p.handleClose(name)

			log.Debug().Str("filter_name", name).Msg("Filter exited")
		}(name, filter, p.channels[name])
	}

	for name, output := range p.Outputs {
		p.channels[name] = make(clogger.MessageChannel, 10)
		p.wg.Add(1)
		go func(name string, output outputs.Outputter, pipe clogger.MessageChannel) {
			defer p.wg.Done()
			outputs.StartOutputter(pipe, output)
			p.handleClose(name)
		}(name, output, p.channels[name])
	}

	if p.debug {
		go func() {
			for {
				outputStr := ""
				for name, pipe := range inputPipes {
					outputStr += fmt.Sprintf("[Input %s %d/%d] ", name, len(pipe), cap(pipe))
				}

				for name, pipe := range p.channels {
					outputStr += fmt.Sprintf("[Filter %s %d/%d] ", name, len(pipe), cap(pipe))
				}

				fmt.Println(outputStr)
				time.Sleep(1 * time.Second)
			}
		}()
	}

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
		filterWg.Wait()

		p.wg.Wait()
	}()
}
