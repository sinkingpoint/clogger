package inputs_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/testutils/mock_inputs"
)

func TestJournalDInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJournalD := mock_inputs.NewMockJournalDReader(ctrl)
	mockJournalD.EXPECT().GetEntry(context.Background()).DoAndReturn(func(ctx context.Context) (inputs.Message, error) {
		return inputs.Message{
			MonoTimestamp: 10,
			RawMessage:    "test",
		}, nil
	}).MinTimes(2)

	flushChan := make(chan []clogger.Message)

	journalDInput := inputs.JournalDInput{
		Reader: mockJournalD,
	}

	go journalDInput.Run(context.Background(), flushChan)

	messages := <-flushChan
	message2 := <-flushChan
	messages = append(messages, message2...)

	if len(messages) != 2 {
		t.Errorf("Expected to fetch two messages, got %d", len(messages))
	}
}
