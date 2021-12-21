package inputs_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/testutils/mock_inputs"
)

func TestJournalDInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJournalD := mock_inputs.NewMockJournalDReader(ctrl)
	mockJournalD.EXPECT().GetEntry().DoAndReturn(func() (inputs.Message, error) {
		return inputs.Message{
			MonoTimestamp: 10,
			RawMessage:    "test",
		}, nil
	}).MinTimes(2)

	flushChan := make(chan clogger.Messages)

	journalDInput := inputs.JournalDInput{
		SendRecvBase: clogger.NewSendRecvBase(clogger.NewSendRecvConfigBase(2, time.Second, flushChan)),
		Reader:       mockJournalD,
	}

	journalDInput.Run(context.Background(), sync.WaitGroup{})

	messages := <-flushChan

	if len(messages) != 2 {
		t.Errorf("Expected to fetch two messages, got %d", len(messages))
	}
}
