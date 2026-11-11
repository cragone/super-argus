// Command argus-agent discovers running workloads on a host and ships their
// logs to Argus. Kubernetes pods (/var/log/pods) appear as soon as they log;
// systemd units (journald) are collected only when named in ARGUS_JOURNAL_UNITS.
package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// parseLabels reads "k=v,k2=v2".
func parseLabels(s string) map[string]string {
	out := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		if k, v, ok := strings.Cut(strings.TrimSpace(kv), "="); ok && k != "" {
			out[k] = v
		}
	}
	return out
}

func parseSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out[v] = true
		}
	}
	return out
}

func detectMode(podsDir string) string {
	if fi, err := os.Stat(podsDir); err == nil && fi.IsDir() {
		return "kubernetes"
	}
	if _, err := exec.LookPath("journalctl"); err == nil {
		return "journal"
	}
	return ""
}

func main() {
	url := env("ARGUS_URL", "http://localhost:8080")
	key := os.Getenv("ARGUS_API_KEY")
	if key == "" {
		log.Fatal("argus-agent: ARGUS_API_KEY is required (create one in Argus: POST /api/v1/api-keys)")
	}
	podsDir := env("ARGUS_PODS_DIR", "/var/log/pods")
	mode := env("ARGUS_AGENT_MODE", "auto")
	if mode == "auto" {
		mode = detectMode(podsDir)
	}
	if mode != "kubernetes" && mode != "journal" {
		log.Fatal("argus-agent: nothing to collect: no " + podsDir + " and no journalctl (set ARGUS_AGENT_MODE=kubernetes|journal)")
	}

	var units []string
	if mode == "journal" {
		for u := range parseSet(os.Getenv("ARGUS_JOURNAL_UNITS")) {
			units = append(units, normalizeUnit(u))
		}
		if len(units) == 0 {
			log.Fatal("argus-agent: journal mode needs ARGUS_JOURNAL_UNITS (comma-separated systemd units to collect)")
		}
		sort.Strings(units)
	}

	stateDir := env("ARGUS_STATE_DIR", "/var/lib/argus-agent")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		stateDir = filepath.Join(os.TempDir(), "argus-agent")
		log.Printf("argus-agent: cannot use state dir, falling back to %s", stateDir)
	}
	st := LoadState(filepath.Join(stateDir, "state.json"))
	hub := NewHub(url, key, parseLabels(os.Getenv("ARGUS_LABELS")))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	liveness := NewLivenessReporter(url, key)
	go liveness.Run(ctx, 15*time.Second)
	switch mode {
	case "kubernetes":
		go runK8sLiveness(ctx, liveness)
	case "journal":
		go runSystemdLiveness(ctx, liveness, 10*time.Second, units)
	}

	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := st.Save(); err != nil {
					log.Printf("argus-agent: save state: %v", err)
				}
			}
		}
	}()

	if env("ARGUS_ACTIONS_ENABLED", "") == "true" {
		if r := newActionRunner(url, key, mode, podsDir); r != nil {
			go r.run(ctx)
		}
	}

	log.Printf("argus-agent: mode=%s url=%s", mode, url)
	switch mode {
	case "kubernetes":
		runKubernetes(ctx, hub, st, podsDir, parseSet(env("ARGUS_EXCLUDE_NAMESPACES", "")))
	case "journal":
		runJournal(ctx, hub, st, units)
	}

	// flush what is queued and remember where we stopped
	flush, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hub.Close(flush)
	if err := st.Save(); err != nil {
		log.Printf("argus-agent: save state: %v", err)
	}
}
