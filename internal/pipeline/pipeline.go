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
	"github.com/sinkingpoint/clogger/internal/metrics"
	"github.com/sinkingpoint/clogger/internal/outputs"
)

type LinkType int

const (
	LINK_TYPE_NORMAL LinkType = iota
	LINK_TYPE_BUFFER
)

type Link struct {
	To   string
	Type LinkType
}

func NewLink(to string) Link {
	return Link{
		To:   to,
		Type: LINK_TYPE_NORMAL,
	}
}

func NewBufferLink(to string) Link {
	return Link{
		To:   to,
		Type: LINK_TYPE_BUFFER,
	}
}

type Pipeline struct {
	Inputs      map[string]inputs.Inputter
	Filters     map[string]filters.Filter
	Outputs     map[string]outputs.Outputter
	Pipes       map[string][]Link
	RevPipes    map[string][]Link
	killChannel chan bool
	debug       bool

	channels   map[string]clogger.MessageChannel
	closedLock sync.Mutex
	closed     map[string]bool

	wg sync.WaitGroup
}

func NewPipeline(inputs map[string]inputs.Inputter, outputs map[string]outputs.Outputter, filters map[string]filters.Filter, pipes map[string][]Link) *Pipeline {
	revPipes := make(map[string][]Link, len(pipes))

	for from, tos := range pipes {
		for _, to := range tos {
			revPipes[to.To] = append(revPipes[to.To], Link{
				To:   from,
				Type: to.Type,
			})
		}
	}

	return &Pipeline{
		Inputs:      inputs,
		Outputs:     outputs,
		Filters:     filters,
		Pipes:       pipes,
		RevPipes:    revPipes,
		debug:       false,
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
			if _, closed := p.closed[dest.To]; closed {
				continue
			}

			for _, src := range p.RevPipes[dest.To] {
				if _, closed := p.closed[src.To]; !closed {
					continue outer
				}
			}

			close(p.channels[dest.To])
			toHandle = append(toHandle, dest.To)
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
	inputWg := sync.WaitGroup{}
	filterWg := sync.WaitGroup{}
	inputCloseChannels := map[string]chan bool{}

	for name, input := range p.Inputs {
		inputCloseChannels[name] = make(chan bool)

		err := input.Init(context.Background())
		if err != nil {
			log.Error().Str("step_name", name).Err(err).Msg("Failed to start input")
			continue
		}

		inputWg.Add(2)
		go func(name string, input inputs.Inputter, killChannel chan bool) {
			defer inputWg.Done()

			ctx, cancel := context.WithCancel(context.Background())
			cancelled := false

			go func() {
				defer inputWg.Done()
				<-killChannel
				close(killChannel)
				cancelled = true
				cancel()
			}()

			for {
				batch, err := input.GetBatch(ctx)

				if err != nil {
					log.Warn().Err(err).Str("step_name", name).Msg("Failed to get batch from input")
					continue
				}

				if batch != nil {
					metrics.MessagesProcessed.WithLabelValues(name, "input").Add(float64(len(batch.Messages)))
					processedLinks := 0

					for _, link := range p.Pipes[name] {
						if processedLinks >= 1 {
							batch = clogger.CloneBatch(batch)
						}
						p.channels[link.To] <- clogger.CloneBatch(batch)
						processedLinks += 1
					}
				}

				if cancelled {
					break
				}
			}

			input.Close(context.Background())
			p.handleClose(name)
		}(name, input, inputCloseChannels[name])
	}

	for name, filter := range p.Filters {
		p.channels[name] = make(clogger.MessageChannel, 10)
		filterWg.Add(1)
		go func(name string, filter filters.Filter, inputPipe clogger.MessageChannel) {
			defer filterWg.Done()
			for batch := range inputPipe {
				currentIndex := 0
				for _, msg := range batch.Messages {
					shouldDrop, err := filter.Filter(context.Background(), &msg)
					if err != nil {
						log.Warn().Err(err).Msg("Filter failed")
					}

					if !shouldDrop {
						batch.Messages[currentIndex] = msg
						currentIndex += 1
					}
				}

				metrics.FilterDropped.WithLabelValues(name).Add(float64(len(batch.Messages) - currentIndex))
				metrics.MessagesProcessed.WithLabelValues(name, "filter").Add(float64(len(batch.Messages)))

				batch.Messages = batch.Messages[:currentIndex]

				processedLinks := 0

				for _, link := range p.Pipes[name] {
					if processedLinks >= 1 {
						batch = clogger.CloneBatch(batch)
					}
					p.channels[link.To] <- clogger.CloneBatch(batch)
					processedLinks += 1
				}
			}
			p.handleClose(name)

			log.Debug().Str("filter_name", name).Msg("Filter exited")
		}(name, filter, p.channels[name])
	}

	for name, output := range p.Outputs {
		if _, ok := p.channels[name]; !ok {
			p.channels[name] = make(clogger.MessageChannel, 10)
		}
		p.wg.Add(1)

		var bufferChannel clogger.MessageChannel

		for _, pipe := range p.Pipes[name] {
			if pipe.Type == LINK_TYPE_BUFFER {
				if _, ok := p.channels[pipe.To]; !ok {
					p.channels[pipe.To] = make(clogger.MessageChannel, 10)
				}

				bufferChannel = p.channels[pipe.To]
			} else {
				log.Panic().Msg("BUG: Found output link that isn't a buffer link")
			}
		}

		go func(name string, output outputs.Outputter, pipe clogger.MessageChannel) {
			defer p.wg.Done()
			outputs.StartOutputter(name, pipe, output, bufferChannel)
			p.handleClose(name)
		}(name, output, p.channels[name])
	}

	if p.debug {
		go func() {
			for {
				outputStr := ""
				for name, pipe := range p.channels {
					outputStr += fmt.Sprintf("[Step %s %d/%d] ", name, len(pipe), cap(pipe))
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
		for name := range p.Inputs {
			inputCloseChannels[name] <- true
		}

		inputWg.Wait()
		filterWg.Wait()
		p.wg.Wait()
	}()
}
