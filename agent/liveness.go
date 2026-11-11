package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// livenessPing mirrors the server's internal/ingest.livenessPing wire shape.
type livenessPing struct {
	Service string `json:"service"`
	Source  string `json:"source"` // "systemd" or "kubernetes"
	Alive   bool   `json:"alive"`
	Detail  string `json:"detail,omitempty"`
}

// LivenessReporter batches per-service liveness pings and POSTs them to Argus
// on an interval. It is separate from Hub: liveness is one small row per
// service (overwritten between flushes, not queued), unlike log lines, so it
// needs none of the shipper's per-stream queueing/backpressure.
type LivenessReporter struct {
	url, key string
	client   *http.Client

	mu      sync.Mutex
	pending map[string]livenessPing
}

func NewLivenessReporter(url, key string) *LivenessReporter {
	return &LivenessReporter{
		url: url, key: key,
		client:  &http.Client{Timeout: 10 * time.Second},
		pending: map[string]livenessPing{},
	}
}

// Report records a service's current liveness, overwriting any pending
// report for the same service, to be sent on the next flush. Collectors call
// this on every poll, not only on change, so Argus can tell "confirmed down"
// apart from "this agent stopped reporting".
func (r *LivenessReporter) Report(service, source string, alive bool, detail string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending[service] = livenessPing{Service: service, Source: source, Alive: alive, Detail: detail}
}

// Run flushes pending pings every interval until ctx ends, then flushes once more.
func (r *LivenessReporter) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			flush, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			r.flush(flush)
			return
		case <-t.C:
			r.flush(ctx)
		}
	}
}

func (r *LivenessReporter) flush(ctx context.Context) {
	r.mu.Lock()
	if len(r.pending) == 0 {
		r.mu.Unlock()
		return
	}
	pings := make([]livenessPing, 0, len(r.pending))
	for _, p := range r.pending {
		pings = append(pings, p)
	}
	r.mu.Unlock()

	body, err := json.Marshal(struct {
		Pings []livenessPing `json:"pings"`
	}{pings})
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url+"/api/v1/ingest/liveness", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Argus-Key", r.key)
	resp, err := r.client.Do(req)
	if err != nil {
		log.Printf("argus-agent: liveness report: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("argus-agent: liveness report: status %d", resp.StatusCode)
	}
}
