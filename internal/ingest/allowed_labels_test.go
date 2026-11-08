package ingest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cragone/Super-Argus/internal/auth"
	"github.com/cragone/Super-Argus/internal/models"
	"github.com/stretchr/testify/require"
)

// pushAs pushes body using a specific plaintext API key, bypassing rig.push's
// hardcoded key.
func (r *rig) pushAs(apiKey, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/push", strings.NewReader(body))
	req.Header.Set(KeyHeader, apiKey)
	rec := httptest.NewRecorder()
	r.h.Push(rec, req)
	return rec
}

func (r *rig) createKey(allowedLabels string) string {
	r.t.Helper()
	plain, prefix, hash, err := auth.GenerateAPIKey()
	require.NoError(r.t, err)
	require.NoError(r.t, r.g.Create(&models.APIKey{Name: "restricted", KeyHash: hash, KeyPrefix: prefix, AllowedLabels: allowedLabels}).Error)
	return plain
}

func TestAllowedLabelsRestrictsIngest(t *testing.T) {
	r := newRig(t)
	cfg, err := json.Marshal(map[string]any{"service": "checkout"})
	require.NoError(t, err)
	restricted := r.createKey(string(cfg))

	rec := r.pushAs(restricted, line(t0, "checkout", "ok"))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var ok pushResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &ok))
	require.Equal(t, 1, ok.Accepted)

	rec = r.pushAs(restricted, line(t0, "other", "nope"))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var bad pushResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &bad))
	require.Equal(t, 1, bad.Rejected)
	require.Contains(t, bad.Errors[0], "not allowed for this API key")

	// A label this key's AllowedLabels does not mention passes through unrestricted.
	cfg2, err := json.Marshal(map[string]any{"env": "prod"})
	require.NoError(t, err)
	other := r.createKey(string(cfg2))
	b, _ := json.Marshal(map[string]any{"ts": t0, "labels": map[string]string{"env": "prod", "service": "anything"}, "line": "ok"})
	rec = r.pushAs(other, string(b)+"\n")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestAllowedLabelsSupportsMultipleValues(t *testing.T) {
	r := newRig(t)
	cfg, err := json.Marshal(map[string]any{"service": []string{"checkout", "billing"}})
	require.NoError(t, err)
	restricted := r.createKey(string(cfg))

	for _, svc := range []string{"checkout", "billing"} {
		rec := r.pushAs(restricted, line(t0, svc, "ok"))
		require.Equal(t, http.StatusOK, rec.Code, svc, rec.Body.String())
	}
	rec := r.pushAs(restricted, line(t0, "search", "nope"))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestParseAllowedLabels(t *testing.T) {
	require.Nil(t, parseAllowedLabels(""))
	require.Nil(t, parseAllowedLabels("{}"))
	require.Nil(t, parseAllowedLabels("not json"))

	got := parseAllowedLabels(`{"service":"checkout"}`)
	require.Equal(t, map[string][]string{"service": {"checkout"}}, got)

	got = parseAllowedLabels(`{"service":["checkout","billing"]}`)
	require.Equal(t, map[string][]string{"service": {"checkout", "billing"}}, got)
}

func TestAllowedLabelsEmptyMeansUnrestricted(t *testing.T) {
	r := newRig(t)
	// r.apiKey was created with AllowedLabels: "{}" in newRig.
	rec := r.push(line(t0, "anything-goes", "ok"), nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
