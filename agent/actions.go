package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Restart actions. The agent polls Argus for restart requests (it never
// listens), runs only what its own allowlist permits, and reports the outcome.
// Disabled unless ARGUS_ACTIONS_ENABLED=true; with an empty allowlist nothing
// is restartable.

const (
	kindSystemd = "systemd_restart"
	kindK8s     = "k8s_restart"

	restartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"
	saDir                 = "/var/run/secrets/kubernetes.io/serviceaccount"
)

// unitName is deliberately strict: it is passed to systemctl.
var unitName = regexp.MustCompile(`^[A-Za-z0-9:_.@\\-]+\.service$`)

type action struct {
	ID      string            `json:"id"`
	Kind    string            `json:"kind"`
	Service string            `json:"service"`
	Target  map[string]string `json:"target"`
}

type actionRunner struct {
	baseURL string
	key     string
	host    string
	http    *http.Client

	// systemd side (journal mode); units is the normalized allowlist.
	units     map[string]bool
	systemctl func(ctx context.Context, unit string) (string, error)

	// kubernetes side (kubernetes mode)
	k8s        *kubeClient
	namespaces map[string]bool
	podsDir    string

	// pause between poll cycles; keeps a skipped action from causing a hot loop.
	pause time.Duration
}

func (r *actionRunner) call(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Argus-Key", r.key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode == http.StatusOK {
		return resp.StatusCode, json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, nil
}

// run polls until ctx ends.
func (r *actionRunner) run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := r.poll(ctx); err != nil && ctx.Err() == nil {
			log.Printf("argus-agent: actions: %v", err)
			sleep(ctx, 5*time.Second)
			continue
		}
		sleep(ctx, r.pause)
	}
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// poll fetches pending actions once and handles those this agent owns.
func (r *actionRunner) poll(ctx context.Context) error {
	var resp struct {
		Items []action `json:"items"`
	}
	code, err := r.call(ctx, http.MethodGet, "/api/v1/agent/actions/pending?host="+url.QueryEscape(r.host), nil, &resp)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("pending: unexpected status %d", code)
	}
	for _, a := range resp.Items {
		r.handle(ctx, a)
	}
	return nil
}

// handle claims and runs one action if this agent is the one that owns it.
func (r *actionRunner) handle(ctx context.Context, a action) {
	var (
		owns bool
		deny string // non-empty: this agent owns it but refuses
		restart func(context.Context) (string, error)
	)
	switch a.Kind {
	case kindSystemd:
		if r.systemctl == nil {
			return
		}
		owns = a.Target["host"] == r.host
		unit := a.Target["unit"]
		switch {
		case !unitName.MatchString(unit):
			deny = "invalid unit name"
		case !r.units[unit]:
			deny = unit + " is not in this agent's restart allowlist"
		}
		restart = func(ctx context.Context) (string, error) { return r.systemctl(ctx, unit) }
	case kindK8s:
		if r.k8s == nil {
			return
		}
		ns, pod := a.Target["namespace"], a.Target["pod"]
		owns = r.podOnThisNode(ns, pod)
		if !r.namespaces[ns] {
			deny = "namespace " + ns + " is not in this agent's restart allowlist"
		}
		restart = func(ctx context.Context) (string, error) { return r.k8s.restartPod(ctx, ns, pod) }
	default:
		return
	}
	if !owns {
		return
	}
	code, err := r.call(ctx, http.MethodPost, "/api/v1/agent/actions/"+url.PathEscape(a.ID)+"/claim", nil, nil)
	if err != nil || code != http.StatusNoContent {
		return // lost the race, or the server is unhappy; the next poll sorts it out
	}
	res := actionResultBody{}
	if deny != "" {
		res.Result = "refused: " + deny
	} else {
		runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		msg, err := restart(runCtx)
		cancel()
		res.OK, res.Result = err == nil, msg
		if err != nil {
			res.Result = err.Error()
		}
	}
	log.Printf("argus-agent: action %s %s %v ok=%v: %s", a.ID, a.Kind, a.Target, res.OK, res.Result)
	// Report with a fresh context so a shutdown right after the restart still records it.
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if _, err := r.call(rctx, http.MethodPost, "/api/v1/agent/actions/"+url.PathEscape(a.ID)+"/result", res, nil); err != nil {
		log.Printf("argus-agent: report action %s: %v", a.ID, err)
	}
}

type actionResultBody struct {
	OK     bool   `json:"ok"`
	Result string `json:"result"`
}

// podOnThisNode reports whether the pod's log directory is under podsDir, i.e.
// the pod runs on this node.
func (r *actionRunner) podOnThisNode(ns, pod string) bool {
	if ns == "" || pod == "" || strings.ContainsAny(ns+pod, `/\_`) {
		return false
	}
	m, _ := filepath.Glob(filepath.Join(r.podsDir, ns+"_"+pod+"_*"))
	return len(m) > 0
}

// runSystemctl restarts a unit via systemctl; the unit was already validated.
func runSystemctl(ctx context.Context, unit string) (string, error) {
	out, err := exec.CommandContext(ctx, "systemctl", "restart", "--", unit).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return msg, fmt.Errorf("systemctl restart %s: %w: %s", unit, err, msg)
	}
	return "restarted " + unit, nil
}

// normalizeUnits maps an allowlist like "nginx,sshd.service" to full unit names.
func normalizeUnits(set map[string]bool) map[string]bool {
	out := map[string]bool{}
	for u := range set {
		out[normalizeUnit(u)] = true
	}
	return out
}

// kubeClient is a minimal in-cluster Kubernetes REST client: just enough to
// roll a workload or delete a bare pod, without pulling in client-go.
type kubeClient struct {
	base  string
	token func() (string, error)
	http  *http.Client
	now   func() time.Time
}

// newInClusterKube builds a client from the pod's service account.
func newInClusterKube() (*kubeClient, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, errors.New("not running in a cluster (KUBERNETES_SERVICE_HOST unset)")
	}
	ca, err := os.ReadFile(filepath.Join(saDir, "ca.crt"))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)
	return &kubeClient{
		base: "https://" + host + ":" + port,
		token: func() (string, error) { // re-read: projected tokens rotate
			b, err := os.ReadFile(filepath.Join(saDir, "token"))
			return strings.TrimSpace(string(b)), err
		},
		http: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}},
		now:  time.Now,
	}, nil
}

type ownerRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type objMeta struct {
	Metadata struct {
		OwnerReferences []ownerRef `json:"ownerReferences"`
	} `json:"metadata"`
}

func (k *kubeClient) do(ctx context.Context, method, path, contentType string, body []byte, out any) error {
	tok, err := k.token()
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, k.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := k.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(b)))
	}
	if out != nil {
		return json.Unmarshal(b, out)
	}
	return nil
}

var workloadPlural = map[string]string{"Deployment": "deployments", "StatefulSet": "statefulsets", "DaemonSet": "daemonsets"}

// restartPod restarts the workload that owns the pod (rolling, via the
// restartedAt annotation, as `kubectl rollout restart` does), or deletes a pod
// that has no owner. Pods owned by anything else (Jobs, operators...) are
// refused rather than guessed at.
func (k *kubeClient) restartPod(ctx context.Context, ns, pod string) (string, error) {
	n, p := url.PathEscape(ns), url.PathEscape(pod)
	var pm objMeta
	if err := k.do(ctx, http.MethodGet, "/api/v1/namespaces/"+n+"/pods/"+p, "", nil, &pm); err != nil {
		return "", err
	}
	owners := pm.Metadata.OwnerReferences
	if len(owners) == 0 {
		if err := k.do(ctx, http.MethodDelete, "/api/v1/namespaces/"+n+"/pods/"+p, "", nil, nil); err != nil {
			return "", err
		}
		return "deleted unmanaged pod " + ns + "/" + pod, nil
	}
	o := owners[0]
	if o.Kind == "ReplicaSet" { // a Deployment's pods are owned via its ReplicaSet
		var rs objMeta
		if err := k.do(ctx, http.MethodGet, "/apis/apps/v1/namespaces/"+n+"/replicasets/"+url.PathEscape(o.Name), "", nil, &rs); err != nil {
			return "", err
		}
		if len(rs.Metadata.OwnerReferences) == 0 {
			return "", fmt.Errorf("replicaset %s has no owning deployment; refusing to guess", o.Name)
		}
		o = rs.Metadata.OwnerReferences[0]
	}
	plural, ok := workloadPlural[o.Kind]
	if !ok {
		return "", fmt.Errorf("pod is owned by %s %s, which this agent does not restart", o.Kind, o.Name)
	}
	patch, _ := json.Marshal(map[string]any{"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{
		"annotations": map[string]string{restartedAtAnnotation: k.now().UTC().Format(time.RFC3339)}}}}})
	path := "/apis/apps/v1/namespaces/" + n + "/" + plural + "/" + url.PathEscape(o.Name)
	if err := k.do(ctx, http.MethodPatch, path, "application/strategic-merge-patch+json", patch, nil); err != nil {
		return "", err
	}
	return fmt.Sprintf("rolling restart of %s %s/%s", strings.ToLower(o.Kind), ns, o.Name), nil
}

// newActionRunner wires restart actions for the agent's mode, or returns nil
// (after logging why) if they cannot be enabled. Allowlists come from
// ARGUS_ACTION_UNITS (journal mode) and ARGUS_ACTION_NAMESPACES (kubernetes
// mode); an empty list means nothing can be restarted.
func newActionRunner(baseURL, key, mode, podsDir string) *actionRunner {
	host := env("ARGUS_ACTION_HOST", "")
	if host == "" {
		host, _ = os.Hostname()
	}
	r := &actionRunner{baseURL: strings.TrimRight(baseURL, "/"), key: key, host: host, podsDir: podsDir,
		http: &http.Client{Timeout: 40 * time.Second}, pause: 2 * time.Second}
	switch mode {
	case "journal":
		r.units = normalizeUnits(parseSet(os.Getenv("ARGUS_ACTION_UNITS")))
		r.systemctl = runSystemctl
		log.Printf("argus-agent: restart actions enabled for host=%s units=%d", host, len(r.units))
	case "kubernetes":
		k, err := newInClusterKube()
		if err != nil {
			log.Printf("argus-agent: restart actions disabled: %v", err)
			return nil
		}
		r.k8s = k
		r.namespaces = parseSet(os.Getenv("ARGUS_ACTION_NAMESPACES"))
		log.Printf("argus-agent: restart actions enabled for namespaces=%d", len(r.namespaces))
	default:
		return nil
	}
	return r
}
