package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("HELMWRAP_CONFIG", `[
		{"action":"transform-values","filter":"$.sops.lastmodified","command":"sops -d {}"},
		{"action":"shell-exec","command":"$HELM"}
	]`)
	_, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig error: %s", err)
	}
}
