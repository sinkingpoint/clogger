SHELL := /bin/bash
VERSION := $(shell git describe --tags --long --always --dirty="-dev")
VERSION_FLAGS := -ldflags='-X "github.com/sinkingpoint/clogger/cmd/clogger/build.GitHash=$(VERSION)"'

.PHONY: commit
commit:
	gotip fmt ./...
	gotip mod tidy

.PHONY: build
build:
	gotip build $(VERSION_FLAGS) ./cmd/clogger 