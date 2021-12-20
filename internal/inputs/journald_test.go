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
	mockJournalD.EXPECT().GetEntry().DoAndReturn(func() (inputs.Message, error) {
		return inputs.Message{
			MonoTimestamp: 0,
			RawMessage:    "test",
		}, nil
	}).MinTimes(2).MaxTimes(2)

	journalDInput := inputs.JournalDInput{
		Reader:    mockJournalD,
		BatchSize: 2,
	}

	dest := clogger.GetMessages(context.Background(), 5)
	defer clogger.PutMessages(context.Background(), dest)
	err := journalDInput.Fetch(context.Background(), dest)
	if err != nil {
		t.Fatalf("Error fetching data: %s", err)
	}

	if len(dest.Messages) != 2 {
		t.Errorf("Expected to fetch two messages, got %d", len(dest.Messages))
	}
}
