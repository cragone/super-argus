package main

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestTUIConfigIsValidYAML(t *testing.T) {
	var cfg struct {
		Version   string `yaml:"version"`
		Processes map[string]struct {
			Command      string `yaml:"command"`
			Availability struct {
				Restart string `yaml:"restart"`
			} `yaml:"availability"`
		} `yaml:"processes"`
	}
	if err := yaml.Unmarshal([]byte(tuiConfig), &cfg); err != nil {
		t.Fatalf("tuiConfig is not valid YAML: %v", err)
	}
	for _, name := range []string{"argus", "argus-agent", "liveness"} {
		p, ok := cfg.Processes[name]
		if !ok {
			t.Fatalf("missing process %q", name)
		}
		if p.Command == "" {
			t.Errorf("process %q has an empty command", name)
		}
		if p.Availability.Restart != "no" {
			t.Errorf("process %q: restart = %q, want \"no\"", name, p.Availability.Restart)
		}
	}
	live := cfg.Processes["liveness"].Command
	for _, want := range []string{"systemctl is-active --quiet", "for u in argus argus-agent", "sleep 5"} {
		if !strings.Contains(live, want) {
			t.Errorf("liveness command missing %q:\n%s", want, live)
		}
	}
}
