package outputs_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/outputs"
	"github.com/sinkingpoint/clogger/internal/outputs/format"
	"github.com/sinkingpoint/clogger/testutils/mock_outputs"
)

// TestSenderFlushesOnFullBuffer tests that when the buffer is full,
// if we queue more messages, then the buffer gets flushed first
func TestSenderFlushesOnFullBuffer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOutput := mock_outputs.NewMockOutputter(ctrl)
	// We expect to flush the buffer exactly once
	mockOutput.EXPECT().FlushToOutput(gomock.Any(), gomock.Any()).Times(1)

	s := outputs.NewSender(outputs.SendConfig{
		FlushInterval: 10 * time.Second,
		BatchSize:     2,
		Formatter:     &format.JSONFormatter{},
	}, mockOutput)

	batch := clogger.GetMessageBatch(2)
	batch.Messages = append(batch.Messages, clogger.NewMessage(), clogger.NewMessage())

	// Fill up the queue
	s.QueueMessages(context.Background(), batch)

	batch = clogger.GetMessageBatch(1)
	batch.Messages = append(batch.Messages, clogger.NewMessage())

	// Try and send another message, which should flush the buffer of the previous two messages
	s.QueueMessages(context.Background(), batch)
}
