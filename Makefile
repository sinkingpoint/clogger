SHELL := /bin/bash
VERSION := $(shell git describe --tags --long --always --dirty="-dev")
VERSION_FLAGS := -ldflags='-X "github.com/sinkingpoint/clogger/cmd/clogger/build.GitHash=$(VERSION)"'

.PHONY: genmocks
genmocks:
	mockgen -source=./internal/inputs/interfaces.go -destination testutils/mock_inputs/inputter.go Inputter
	mockgen -source=./internal/inputs/journald.go -destination testutils/mock_inputs/journald.go JournalDReader
	mockgen -source=./internal/outputs/interfaces.go -destination testutils/mock_outputs/outputter.go Outputter


.PHONY: test
test:
	gotip test ./...

.PHONY: commit
commit: test
	gotip fmt ./...
	gotip mod tidy

.PHONY: build
build:
	gotip build $(VERSION_FLAGS) ./cmd/clogger