package filters_test

import (
	"context"
	"testing"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/filters"
)

func TestRateLimitFilter(t *testing.T) {
	filter := filters.NewRateLimitFilter(filters.RateLimitFilterConfig{
		PartitionKey: "test",
		Rate:         1,
	})

	if shouldDrop, _ := filter.Filter(context.Background(), &clogger.Message{
		MonoTimestamp: time.Now().UnixNano(),
		ParsedFields: map[string]interface{}{
			"test": "a",
		},
	}); shouldDrop {
		t.Fatal("Message got filtered when it shouldn't have")
	}

	// Uses the same key as above so should fail
	if shouldDrop, _ := filter.Filter(context.Background(), &clogger.Message{
		MonoTimestamp: time.Now().UnixNano(),
		ParsedFields: map[string]interface{}{
			"test": "a",
		},
	}); !shouldDrop {
		t.Fatal("Message didn't get filtered when it should have")
	}

	// Uses a different key, so should pass
	if shouldDrop, _ := filter.Filter(context.Background(), &clogger.Message{
		MonoTimestamp: time.Now().UnixNano(),
		ParsedFields: map[string]interface{}{
			"test": "b",
		},
	}); shouldDrop {
		t.Fatal("Message didn't get filtered when it should have")
	}
}
