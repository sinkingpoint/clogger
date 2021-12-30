package clogger_test

import (
	"testing"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

func TestGetMessageBatch(t *testing.T) {
	batch := clogger.GetMessageBatch(10)
	if batch == nil || cap(batch.Messages) < 10 {
		t.Fatal("Failed to get batch of size 10")
	}

	if len(batch.Messages) != 0 {
		t.Fatal("Batch wasn't reset")
	}

	clogger.PutMessageBatch(batch)

	batch = clogger.GetMessageBatch(12)

	if batch == nil || cap(batch.Messages) < 12 {
		t.Fatalf("Failed to get batch of size 12 - got size %d", cap(batch.Messages))
	}

	if len(batch.Messages) != 0 {
		t.Fatal("Batch wasn't reset")
	}

	clogger.PutMessageBatch(batch)
}
