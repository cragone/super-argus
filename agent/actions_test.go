package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// assertions is a tiny stand-in for testify: this module has no dependencies.
type assertions struct{}

var require assertions

func msg(m []any) string {
	if len(m) == 0 {
		return ""
	}
	return fmt.Sprint(m...)
}

func (assertions) NoError(t *testing.T, err error, m ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v %s", err, msg(m))
	}
}

func (assertions) True(t *testing.T, v bool, m ...any) {
	t.Helper()
	if !v {
		t.Fatalf("expected true %s", msg(m))
	}
}

func (assertions) False(t *testing.T, v bool, m ...any) {
	t.Helper()
	if v {
		t.Fatalf("expected false %s", msg(m))
	}
}

func (assertions) Equal(t *testing.T, want, got any, m ...any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %#v, got %#v %s", want, got, msg(m))
	}
}

func has(hay any, needle string) bool {
	switch h := hay.(type) {
	case string:
		return strings.Contains(h, needle)
	case []string:
		for _, v := range h {
			if v == needle {
				return true
			}
		}
	}
	return false
}

func (assertions) Contains(t *testing.T, hay any, needle string, m ...any) {
	t.Helper()
	if !has(hay, needle) {
		t.Fatalf("%v does not contain %q %s", hay, needle, msg(m))
	}
}

func (assertions) NotContains(t *testing.T, hay any, needle string, m ...any) {
	t.Helper()
	if has(hay, needle) {
		t.Fatalf("%v unexpectedly contains %q %s", hay, needle, msg(m))
	}
}

// fakeArgus serves one batch of pending actions and records claims and results.
type fakeArgus struct {
	mu      sync.Mutex
	items   []action
	claimed []string
	results map[string]actionResultBody
	srv     *httptest.Server
}

func newFakeArgus(t *testing.T, items ...action) *fakeArgus {
	f := &fakeArgus{items: items, results: map[string]actionResultBody{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("X-Argus-Key") != "k" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/agent/actions/pending":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": f.items})
		case strings.HasSuffix(r.URL.Path, "/claim"):
			f.claimed = append(f.claimed, strings.Split(r.URL.Path, "/")[5])
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(r.URL.Path, "/result"):
			var b actionResultBody
			_ = json.NewDecoder(r.Body).Decode(&b)
			f.results[strings.Split(r.URL.Path, "/")[5]] = b
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeArgus) runner(t *testing.T) *actionRunner {
	return &actionRunner{baseURL: f.srv.URL, key: "k", host: "h1", http: f.srv.Client(), pause: time.Millisecond}
}

func TestSystemdRestartAllowlist(t *testing.T) {
	f := newFakeArgus(t,
		action{ID: "ok", Kind: kindSystemd, Target: map[string]string{"host": "h1", "unit": "nginx.service"}},
		action{ID: "denied", Kind: kindSystemd, Target: map[string]string{"host": "h1", "unit": "sshd.service"}},
		action{ID: "bad", Kind: kindSystemd, Target: map[string]string{"host": "h1", "unit": "x; rm -rf /"}},
		action{ID: "otherhost", Kind: kindSystemd, Target: map[string]string{"host": "h2", "unit": "nginx.service"}},
	)
	r := f.runner(t)
	r.units = normalizeUnits(map[string]bool{"nginx": true})
	var ran []string
	r.systemctl = func(_ context.Context, unit string) (string, error) {
		ran = append(ran, unit)
		return "restarted " + unit, nil
	}

	require.NoError(t, r.poll(context.Background()))

	require.Equal(t, []string{"nginx.service"}, ran, "only the allowlisted unit on this host runs")
	require.True(t, f.results["ok"].OK)
	require.False(t, f.results["denied"].OK)
	require.Contains(t, f.results["denied"].Result, "allowlist")
	require.False(t, f.results["bad"].OK)
	require.NotContains(t, f.claimed, "otherhost", "another host's action is not claimed")
}

func TestSystemdFailureIsReported(t *testing.T) {
	f := newFakeArgus(t, action{ID: "a", Kind: kindSystemd, Target: map[string]string{"host": "h1", "unit": "nginx.service"}})
	r := f.runner(t)
	r.units = map[string]bool{"nginx.service": true}
	r.systemctl = func(context.Context, string) (string, error) { return "", errors.New("unit not found") }
	require.NoError(t, r.poll(context.Background()))
	require.False(t, f.results["a"].OK)
	require.Contains(t, f.results["a"].Result, "unit not found")
}

func TestEmptyAllowlistRestartsNothing(t *testing.T) {
	f := newFakeArgus(t, action{ID: "a", Kind: kindSystemd, Target: map[string]string{"host": "h1", "unit": "nginx.service"}})
	r := f.runner(t)
	r.systemctl = func(context.Context, string) (string, error) { t.Fatal("must not run"); return "", nil }
	require.NoError(t, r.poll(context.Background()))
	require.False(t, f.results["a"].OK)
}

// fakeKube answers the handful of API calls restartPod makes.
type fakeKube struct {
	pods     map[string][]ownerRef // "ns/pod" -> owners
	replsets map[string]ownerRef   // "ns/rs" -> owner
	calls    []string
	patch    string
}

func (k *fakeKube) handler(w http.ResponseWriter, r *http.Request) {
	k.calls = append(k.calls, r.Method+" "+r.URL.Path)
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	obj := func(owners ...ownerRef) {
		var m objMeta
		m.Metadata.OwnerReferences = owners
		_ = json.NewEncoder(w).Encode(m)
	}
	switch {
	case r.Method == http.MethodGet && len(p) == 6 && p[0] == "api" && p[4] == "pods": // /api/v1/namespaces/ns/pods/name
		owners, ok := k.pods[p[3]+"/"+p[5]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		obj(owners...)
	case r.Method == http.MethodGet && p[0] == "apis" && p[5] == "replicasets":
		if o, ok := k.replsets[p[4]+"/"+p[6]]; ok {
			obj(o)
		} else {
			obj()
		}
	case r.Method == http.MethodPatch:
		b, _ := io.ReadAll(r.Body)
		k.patch = r.Header.Get("Content-Type") + " " + string(b)
		obj()
	case r.Method == http.MethodDelete:
		obj()
	default:
		http.NotFound(w, r)
	}
}

func newKubeRunner(t *testing.T, f *fakeArgus, fk *fakeKube) *actionRunner {
	ks := httptest.NewServer(http.HandlerFunc(fk.handler))
	t.Cleanup(ks.Close)
	podsDir := t.TempDir()
	for _, d := range []string{"prod_web-7d9f8b6c5d-x2k4p_uid1", "prod_batch-1_uid2", "prod_solo_uid3", "prod_job-x_uid4", "kube-system_dns_uid5"} {
		require.NoError(t, os.MkdirAll(filepath.Join(podsDir, d), 0o755))
	}
	r := f.runner(t)
	r.podsDir = podsDir
	r.namespaces = map[string]bool{"prod": true}
	r.k8s = &kubeClient{base: ks.URL, token: func() (string, error) { return "tok", nil }, http: ks.Client(),
		now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }}
	return r
}

func TestKubernetesRestart(t *testing.T) {
	podAct := func(id, ns, pod string) action {
		return action{ID: id, Kind: kindK8s, Target: map[string]string{"namespace": ns, "pod": pod}}
	}
	f := newFakeArgus(t,
		podAct("deploy", "prod", "web-7d9f8b6c5d-x2k4p"),
		podAct("sts", "prod", "batch-1"),
		podAct("bare", "prod", "solo"),
		podAct("job", "prod", "job-x"),
		podAct("denied", "kube-system", "dns"),
		podAct("elsewhere", "prod", "not-on-this-node"),
	)
	fk := &fakeKube{
		pods: map[string][]ownerRef{
			"prod/web-7d9f8b6c5d-x2k4p": {{Kind: "ReplicaSet", Name: "web-7d9f8b6c5d"}},
			"prod/batch-1":              {{Kind: "StatefulSet", Name: "batch"}},
			"prod/solo":                 nil,
			"prod/job-x":                {{Kind: "Job", Name: "job"}},
			"kube-system/dns":           {{Kind: "ReplicaSet", Name: "dns-1"}},
		},
		replsets: map[string]ownerRef{"prod/web-7d9f8b6c5d": {Kind: "Deployment", Name: "web"}},
	}
	r := newKubeRunner(t, f, fk)
	require.NoError(t, r.poll(context.Background()))

	require.True(t, f.results["deploy"].OK, f.results["deploy"].Result)
	require.Contains(t, f.results["deploy"].Result, "deployment prod/web")
	require.Contains(t, fk.calls, "PATCH /apis/apps/v1/namespaces/prod/deployments/web")
	require.Contains(t, fk.patch, "application/strategic-merge-patch+json")
	require.Contains(t, fk.patch, `"kubectl.kubernetes.io/restartedAt":"2026-01-02T03:04:05Z"`)

	require.True(t, f.results["sts"].OK)
	require.Contains(t, fk.calls, "PATCH /apis/apps/v1/namespaces/prod/statefulsets/batch")

	require.True(t, f.results["bare"].OK)
	require.Contains(t, fk.calls, "DELETE /api/v1/namespaces/prod/pods/solo")

	require.False(t, f.results["job"].OK, "unsupported owner kinds fail closed")
	require.Contains(t, f.results["job"].Result, "Job")

	require.False(t, f.results["denied"].OK)
	require.Contains(t, f.results["denied"].Result, "allowlist")
	for _, c := range fk.calls {
		require.NotContains(t, c, "kube-system", "a refused namespace is never touched")
	}
	require.NotContains(t, f.claimed, "elsewhere", "pods on other nodes are left to their node's agent")
}
