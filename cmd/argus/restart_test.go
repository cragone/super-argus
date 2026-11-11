package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeServer scripts the endpoints argus restart uses. statuses are returned by
// successive GET /actions/{id} calls.
type fakeServer struct {
	targets  []restartTarget
	statuses []restartAction
	posted   []map[string]string
	polls    int
}

func (f *fakeServer) client(t *testing.T, stdin string) (*restartClient, *bytes.Buffer) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" && r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, `{"error":{"code":"unauthorized","message":"no token"}}`, http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/api/v1/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
		case strings.HasSuffix(r.URL.Path, "/restart-targets"):
			_ = json.NewEncoder(w).Encode(map[string]any{"items": f.targets})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/restart"):
			var in struct{ Labels map[string]string }
			_ = json.NewDecoder(r.Body).Decode(&in)
			f.posted = append(f.posted, in.Labels)
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(restartAction{ID: "a1", Status: "pending"})
		case strings.HasPrefix(r.URL.Path, "/api/v1/actions/"):
			s := f.statuses[min(f.polls, len(f.statuses)-1)]
			f.polls++
			_ = json.NewEncoder(w).Encode(s)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	out := &bytes.Buffer{}
	c := &restartClient{base: srv.URL, http: srv.Client(), in: strings.NewReader(stdin), out: out, poll: time.Millisecond, wait: 2 * time.Second}
	require.NoError(t, c.login(context.Background(), "a@x.io", "pw"))
	return c, out
}

var nginx = restartTarget{Kind: "systemd_restart", Target: map[string]string{"host": "h1", "unit": "nginx.service"},
	Labels: map[string]string{"source": "journald", "host": "h1", "unit": "nginx.service"}}
var webPod = restartTarget{Kind: "k8s_restart", Target: map[string]string{"namespace": "prod", "pod": "web-1"},
	Labels: map[string]string{"source": "kubernetes", "namespace": "prod", "pod": "web-1"}}

func TestRestartSucceedsAfterPolling(t *testing.T) {
	f := &fakeServer{targets: []restartTarget{nginx}, statuses: []restartAction{
		{ID: "a1", Status: "running"}, {ID: "a1", Status: "succeeded", Result: "restarted nginx.service"}}}
	c, out := f.client(t, "y\n")
	require.NoError(t, c.restart(context.Background(), "nginx", 0, false))
	require.Equal(t, []map[string]string{nginx.Labels}, f.posted)
	require.Contains(t, out.String(), "Restart nginx.service on h1? [y/N]")
	require.Contains(t, out.String(), "done: restarted nginx.service")
}

func TestRestartDeclinedSendsNothing(t *testing.T) {
	f := &fakeServer{targets: []restartTarget{nginx}}
	c, _ := f.client(t, "n\n")
	require.EqualError(t, c.restart(context.Background(), "nginx", 0, false), "aborted")
	require.Empty(t, f.posted)
}

func TestRestartYesSkipsPrompt(t *testing.T) {
	f := &fakeServer{targets: []restartTarget{webPod}, statuses: []restartAction{{ID: "a1", Status: "succeeded", Result: "ok"}}}
	c, out := f.client(t, "")
	require.NoError(t, c.restart(context.Background(), "web", 0, true))
	require.NotContains(t, out.String(), "[y/N]")
}

func TestRestartAmbiguousNeedsTarget(t *testing.T) {
	f := &fakeServer{targets: []restartTarget{nginx, webPod}, statuses: []restartAction{{ID: "a1", Status: "succeeded"}}}
	c, out := f.client(t, "")
	require.EqualError(t, c.restart(context.Background(), "x", 0, true), "ambiguous target")
	require.Contains(t, out.String(), "2. pod prod/web-1")
	require.Empty(t, f.posted)

	require.Error(t, c.restart(context.Background(), "x", 3, true))
	require.NoError(t, c.restart(context.Background(), "x", 2, true))
	require.Equal(t, []map[string]string{webPod.Labels}, f.posted)
}

func TestRestartReportsFailureAndExpiry(t *testing.T) {
	for _, st := range []restartAction{{Status: "failed", Result: "authorization required"}, {Status: "expired"}} {
		f := &fakeServer{targets: []restartTarget{nginx}, statuses: []restartAction{st}}
		c, _ := f.client(t, "")
		err := c.restart(context.Background(), "nginx", 0, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), st.Status)
	}
}

func TestRestartNoTargets(t *testing.T) {
	f := &fakeServer{}
	c, _ := f.client(t, "")
	err := c.restart(context.Background(), "ghost", 0, true)
	require.ErrorContains(t, err, "no restartable target")
}

func TestRestartTimesOut(t *testing.T) {
	f := &fakeServer{targets: []restartTarget{nginx}, statuses: []restartAction{{ID: "a1", Status: "pending"}}}
	c, _ := f.client(t, "")
	c.wait = 20 * time.Millisecond
	require.ErrorContains(t, c.restart(context.Background(), "nginx", 0, true), "timed out")
}
