package config_test

import (
	"testing"

	"github.com/sinkingpoint/clogger/cmd/clogger/config"
	"gopkg.in/yaml.v3"
)

func TestRawConfigLoad(t *testing.T) {
	raw := `
    type: "journald"
    url: "/run/systemd/journald.sock"
    batch_size: 10
`

	config := config.RawInputConfig{}

	err := yaml.Unmarshal([]byte(raw), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal input config: %s", err)
	}

	if config.Type != "journald" {
		t.Fatalf("Invalid type. Expected 'journald' got '%s'", config.Type)
	}

	if config.Config["batch_size"] != 10 {
		t.Fatalf("Invalid batch size: %d", config.Config["batch_size"])
	}
}
