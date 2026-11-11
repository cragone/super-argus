package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cragone/Super-Argus/internal/auth"
	"github.com/cragone/Super-Argus/internal/models"
	"github.com/stretchr/testify/require"
)

// agentDo calls an agent endpoint with an API key header.
func (e *env) agentDo(method, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("X-Argus-Key", key)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *env) agentKey(name string) string {
	key, prefix, hash, err := auth.GenerateAPIKey()
	require.NoError(e.t, err)
	require.NoError(e.t, e.srv.DB.Create(&models.APIKey{Name: name, KeyHash: hash, KeyPrefix: prefix}).Error)
	return key
}

func TestRestartRequiresAdmin(t *testing.T) {
	e := newEnv(t)
	body := map[string]any{"labels": map[string]string{"source": "journald", "host": "h1", "unit": "nginx.service"}}
	for _, role := range []string{models.RoleViewer, models.RoleEditor} {
		require.Equal(t, http.StatusForbidden, e.do("POST", "/api/v1/services/nginx/restart", e.userToken(role), body).Code, role)
	}
	require.Equal(t, http.StatusUnauthorized, e.do("POST", "/api/v1/services/nginx/restart", "", body).Code)
	var n int64
	e.srv.DB.Model(&models.Action{}).Count(&n)
	require.Zero(t, n)
}

func TestRestartRejectsBadTargets(t *testing.T) {
	e := newEnv(t)
	admin := e.userToken(models.RoleAdmin)
	for name, labels := range map[string]map[string]string{
		"no source":      {"host": "h1"},
		"journald/host":  {"source": "journald"},
		"k8s/no pod":     {"source": "kubernetes", "namespace": "prod"},
		"unknown source": {"source": "docker"},
	} {
		rec := e.do("POST", "/api/v1/services/x/restart", admin, map[string]any{"labels": labels})
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
}

func TestRestartLifecycle(t *testing.T) {
	e := newEnv(t)
	admin := e.userToken(models.RoleAdmin)
	key := e.agentKey("agent-h1")
	otherKey := e.agentKey("agent-h2")

	rec := e.do("POST", "/api/v1/services/nginx/restart", admin,
		map[string]any{"labels": map[string]string{"source": "journald", "host": "h1", "unit": "nginx.service"}})
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	v := decodeBody[actionView](t, rec.Body.Bytes())
	require.Equal(t, models.ActionSystemdRestart, v.Kind)
	require.Equal(t, models.ActionPending, v.Status)
	require.Equal(t, "nginx.service", v.Target["unit"])

	// A repeat while pending is refused.
	rec = e.do("POST", "/api/v1/services/nginx/restart", admin,
		map[string]any{"labels": map[string]string{"source": "journald", "host": "h1", "unit": "nginx.service"}})
	require.Equal(t, http.StatusConflict, rec.Code)

	// Agents need a valid key.
	require.Equal(t, http.StatusUnauthorized, e.agentDo("GET", "/api/v1/agent/actions/pending?host=h1", "", "").Code)
	require.Equal(t, http.StatusUnauthorized, e.agentDo("POST", "/api/v1/agent/actions/"+v.ID+"/claim", "argus_bogus", "").Code)

	// The matching host sees it; another host's poll would not (checked via claim ordering below).
	rec = e.agentDo("GET", "/api/v1/agent/actions/pending?host=h1", key, "")
	require.Equal(t, http.StatusOK, rec.Code)
	pending := decodeBody[struct{ Items []actionView }](t, rec.Body.Bytes())
	require.Len(t, pending.Items, 1)

	require.Equal(t, http.StatusNoContent, e.agentDo("POST", "/api/v1/agent/actions/"+v.ID+"/claim", key, "").Code)
	// Second claim loses the race.
	require.Equal(t, http.StatusConflict, e.agentDo("POST", "/api/v1/agent/actions/"+v.ID+"/claim", otherKey, "").Code)
	// Only the claimer can report.
	require.Equal(t, http.StatusConflict, e.agentDo("POST", "/api/v1/agent/actions/"+v.ID+"/result", otherKey, `{"ok":true}`).Code)
	require.Equal(t, http.StatusNoContent, e.agentDo("POST", "/api/v1/agent/actions/"+v.ID+"/result", key, `{"ok":true,"result":"restarted"}`).Code)

	rec = e.do("GET", "/api/v1/actions/"+v.ID, admin, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	got := decodeBody[actionView](t, rec.Body.Bytes())
	require.Equal(t, models.ActionSucceeded, got.Status)
	require.Equal(t, "restarted", got.Result)
	require.NotEmpty(t, got.RequestedBy)

	rec = e.do("GET", "/api/v1/actions", admin, nil)
	require.Contains(t, rec.Body.String(), v.ID)
	// History is admin-only too.
	require.Equal(t, http.StatusForbidden, e.do("GET", "/api/v1/actions", e.userToken(models.RoleViewer), nil).Code)
}

func TestPendingFiltersSystemdByHost(t *testing.T) {
	e := newEnv(t)
	admin := e.userToken(models.RoleAdmin)
	key := e.agentKey("agent")
	require.Equal(t, http.StatusAccepted, e.do("POST", "/api/v1/services/nginx/restart", admin,
		map[string]any{"labels": map[string]string{"source": "journald", "host": "h1"}}).Code)
	// A k8s action is visible to every agent, so the poll returns immediately even for the wrong host.
	require.Equal(t, http.StatusAccepted, e.do("POST", "/api/v1/services/web/restart", admin,
		map[string]any{"labels": map[string]string{"source": "kubernetes", "namespace": "prod", "pod": "web-abc-1"}}).Code)

	rec := e.agentDo("GET", "/api/v1/agent/actions/pending?host=h2", key, "")
	pending := decodeBody[struct{ Items []actionView }](t, rec.Body.Bytes())
	require.Len(t, pending.Items, 1)
	require.Equal(t, models.ActionK8sRestart, pending.Items[0].Kind)
}

func (e *env) addStream(id uint64, lastSeen time.Time, labels map[string]string) {
	b, _ := json.Marshal(labels)
	require.NoError(e.t, e.srv.DB.Create(&models.Stream{ID: id, Labels: string(b), LabelsText: string(b), FirstSeen: lastSeen, LastSeen: lastSeen}).Error)
	for k, v := range labels {
		require.NoError(e.t, e.srv.DB.Create(&models.StreamLabel{StreamID: id, Key: k, Value: v}).Error)
	}
}

func TestRestartTargets(t *testing.T) {
	e := newEnv(t)
	admin := e.userToken(models.RoleAdmin)
	now := time.Now().UTC()
	e.addStream(1, now, map[string]string{"service": "nginx", "source": "journald", "host": "h1", "unit": "nginx.service"})
	// A second stream of the same unit (extra label) must not duplicate the target.
	e.addStream(2, now, map[string]string{"service": "nginx", "source": "journald", "host": "h1", "unit": "nginx.service", "env": "prod"})
	e.addStream(3, now, map[string]string{"service": "web", "source": "kubernetes", "namespace": "prod", "pod": "web-abc-1", "container": "app"})
	e.addStream(4, now.Add(-time.Hour), map[string]string{"service": "web", "source": "kubernetes", "namespace": "prod", "pod": "web-old-1", "container": "app"})
	e.addStream(5, now, map[string]string{"service": "plain"}) // no source: not restartable

	get := func(svc string) []targetView {
		rec := e.do("GET", "/api/v1/services/"+svc+"/restart-targets", admin, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		return decodeBody[struct{ Items []targetView }](t, rec.Body.Bytes()).Items
	}
	ng := get("nginx")
	require.Len(t, ng, 1)
	require.Equal(t, models.ActionSystemdRestart, ng[0].Kind)
	require.Equal(t, map[string]string{"source": "journald", "host": "h1", "unit": "nginx.service"}, ng[0].Labels)

	web := get("web")
	require.Len(t, web, 1, "the stale pod is left out")
	require.Equal(t, "web-abc-1", web[0].Target["pod"])
	require.Empty(t, get("plain"))

	// What the endpoint returns is accepted by the restart endpoint.
	rec := e.do("POST", "/api/v1/services/web/restart", admin, map[string]any{"labels": web[0].Labels})
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())

	require.Equal(t, http.StatusForbidden, e.do("GET", "/api/v1/services/nginx/restart-targets", e.userToken(models.RoleViewer), nil).Code)
}
