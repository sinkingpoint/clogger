package pipeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/filters"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
	"github.com/sinkingpoint/clogger/internal/pipeline"
	"github.com/sinkingpoint/clogger/testutils/mock_inputs"
	"github.com/sinkingpoint/clogger/testutils/mock_outputs"
)

func TestPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInput := mock_inputs.NewMockInputter(ctrl)
	mockInput.EXPECT().Kill().Times(1)
	mockInput.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, input clogger.MessageChannel) error {
		batch := clogger.GetMessageBatch(3)
		batch.Messages = append(batch.Messages, []clogger.Message{
			{
				MonoTimestamp: 0,
			},
			{
				MonoTimestamp: 1,
			},
			{
				MonoTimestamp: 2,
			},
		}...)

		input <- batch

		return nil
	}).MaxTimes(1)

	mockOutput := mock_outputs.NewMockOutputter(ctrl)
	mockOutput.EXPECT().GetSendConfig().Return(outputs.SendConfig{
		FlushInterval: time.Millisecond * 100,
		BatchSize:     3,
	}).Times(1)

	mockOutput.EXPECT().FlushToOutput(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, batch *clogger.MessageBatch) (outputs.OutputResult, error) {
		if len(batch.Messages) != 3 {
			t.Errorf("Buffer wasn't completly flushed - expected 3 messages, got %d", len(batch.Messages))
		}
		return outputs.OUTPUT_SUCCESS, nil
	}).Times(1)

	pipeline := pipeline.NewPipeline(map[string]inputs.Inputter{
		"test_input": mockInput,
	}, map[string]outputs.Outputter{
		"test_output": mockOutput,
	}, map[string]filters.Filter{}, map[string][]string{
		"test_input": {"test_output"},
	})

	pipeline.Run()

	// Wait for a bit to let the pipeline chug
	pipeline.Kill()
}
