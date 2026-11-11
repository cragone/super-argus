package ingest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cragone/Super-Argus/internal/models"
	"github.com/stretchr/testify/require"
)

func (r *rig) liveness(body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/liveness", strings.NewReader(body))
	req.Header.Set(KeyHeader, r.apiKey)
	rec := httptest.NewRecorder()
	r.h.Liveness(rec, req)
	return rec
}

func TestLivenessUpsertsLatestState(t *testing.T) {
	r := newRig(t)
	rec := r.liveness(`{"pings":[{"service":"nginx","source":"systemd","alive":true}]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp livenessResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 1, resp.Accepted)
	require.Equal(t, 0, resp.Rejected)

	var row models.ServiceLiveness
	require.NoError(t, r.g.Where("service = ?", "nginx").First(&row).Error)
	require.Equal(t, models.LivenessAlive, row.State)

	// A later report for the same service overwrites, it does not duplicate.
	rec = r.liveness(`{"pings":[{"service":"nginx","source":"systemd","alive":false,"detail":"crashed"}]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, r.g.Where("service = ?", "nginx").First(&row).Error)
	require.Equal(t, models.LivenessDown, row.State)
	require.Equal(t, "crashed", row.Detail)
}

func TestLivenessRejectsInvalidPing(t *testing.T) {
	r := newRig(t)
	rec := r.liveness(`{"pings":[{"service":"","source":"systemd","alive":true},{"service":"x","source":"bogus","alive":true}]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp livenessResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Accepted)
	require.Equal(t, 2, resp.Rejected)
}

func TestLivenessRequiresAPIKey(t *testing.T) {
	r := newRig(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/liveness", strings.NewReader(`{"pings":[]}`))
	rec := httptest.NewRecorder()
	r.h.Liveness(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSweeperMarksStaleHeartbeatsUnknown(t *testing.T) {
	r := newRig(t)
	require.NoError(r.t, r.g.Create(&models.ServiceLiveness{
		Service: "nginx", Source: "systemd", State: models.LivenessAlive,
		LastHeartbeat: r.now.Add(-time.Minute), UpdatedAt: r.now.Add(-time.Minute),
	}).Error)
	require.NoError(r.t, r.g.Create(&models.ServiceLiveness{
		Service: "fresh", Source: "systemd", State: models.LivenessAlive,
		LastHeartbeat: r.now, UpdatedAt: r.now,
	}).Error)

	sw := &Sweeper{DB: r.g, Stale: 45 * time.Second, Now: func() time.Time { return r.now }}
	sw.Tick()

	var stale, fresh models.ServiceLiveness
	require.NoError(t, r.g.Where("service = ?", "nginx").First(&stale).Error)
	require.Equal(t, models.LivenessUnknown, stale.State)
	require.NoError(t, r.g.Where("service = ?", "fresh").First(&fresh).Error)
	require.Equal(t, models.LivenessAlive, fresh.State)
}
