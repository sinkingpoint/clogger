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
	mockJournalD.EXPECT().GetEntry(context.Background()).DoAndReturn(func(ctx context.Context) (clogger.Message, error) {
		return clogger.Message{
			MonoTimestamp: 10,
		}, nil
	}).MinTimes(2)

	flushChan := make(clogger.MessageChannel)

	journalDInput, _ := inputs.NewJournalDInputWithReader(inputs.RecvConfig{}, mockJournalD)

	go func() {
		for {
			batch, _ := journalDInput.GetBatch(context.Background())
			flushChan <- batch
		}
	}()

	batch := []clogger.Message{}
	messages := <-flushChan
	message2 := <-flushChan
	batch = append(batch, messages.Messages...)
	batch = append(batch, message2.Messages...)

	clogger.PutMessageBatch(messages)
	clogger.PutMessageBatch(message2)

	if len(batch) != 2 {
		t.Errorf("Expected to fetch two messages, got %d", len(batch))
	}
}
