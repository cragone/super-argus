package ingest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cragone/Super-Argus/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxLivenessBody bounds a liveness report: it carries one row per service an
// agent knows about, nowhere near the size of a log push.
const maxLivenessBody = 1 << 20

// livenessPing is one service's liveness as seen by an agent's collector.
type livenessPing struct {
	Service string `json:"service"`
	Source  string `json:"source"` // "systemd" or "kubernetes"
	Alive   bool   `json:"alive"`
	Detail  string `json:"detail,omitempty"`
}

type livenessBody struct {
	Pings []livenessPing `json:"pings"`
}

type livenessResponse struct {
	Accepted int      `json:"accepted"`
	Rejected int      `json:"rejected"`
	Errors   []string `json:"errors,omitempty"`
}

func validLivenessSource(s string) bool {
	return s == "systemd" || s == "kubernetes"
}

// Liveness ingests a batch of per-service liveness heartbeats from an agent's
// systemd or Kubernetes collector: POST /api/v1/ingest/liveness,
// {"pings":[{"service":"nginx","source":"systemd","alive":true}, ...]}. Agents
// send a heartbeat on every poll, not only on change, so the server can tell
// "confirmed down" apart from "stopped reporting" (see Sweeper).
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	// Liveness reports are not scoped by allowed_labels; any valid key may report any service.
	if _, ok := h.authenticate(r); !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "missing, invalid or revoked API key")
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxLivenessBody)
	var in livenessBody
	if err := json.NewDecoder(body).Decode(&in); err != nil {
		if err == io.EOF {
			writeErr(w, http.StatusBadRequest, "bad_request", "empty body")
			return
		}
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON: "+err.Error())
		return
	}

	now := h.Now()
	var resp livenessResponse
	rows := make([]models.ServiceLiveness, 0, len(in.Pings))
	for _, p := range in.Pings {
		if p.Service == "" || !validLivenessSource(p.Source) {
			resp.Rejected++
			if len(resp.Errors) < maxErrorsEcho {
				resp.Errors = append(resp.Errors, "invalid ping: service and a valid source are required")
			}
			continue
		}
		state := models.LivenessDown
		if p.Alive {
			state = models.LivenessAlive
		}
		rows = append(rows, models.ServiceLiveness{
			Service: p.Service, Source: p.Source, State: state, Detail: p.Detail,
			LastHeartbeat: now, UpdatedAt: now,
		})
	}
	if len(rows) > 0 {
		err := h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "service"}},
			DoUpdates: clause.AssignmentColumns([]string{"source", "state", "detail", "last_heartbeat", "updated_at"}),
		}).Create(&rows).Error
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal", "could not store liveness")
			return
		}
	}
	resp.Accepted = len(rows)
	writeJSON(w, http.StatusOK, resp)
}

// Sweeper flips a service's liveness to "unknown" once its last heartbeat is
// older than Stale, so a dead agent or host shows up instead of leaving a
// stale "alive"/"down" state displayed forever.
type Sweeper struct {
	DB    *gorm.DB
	Stale time.Duration // default 45s (3x the agent's ~15s heartbeat)
	Now   func() time.Time
}

func (s *Sweeper) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Sweeper) stale() time.Duration {
	if s.Stale > 0 {
		return s.Stale
	}
	return 45 * time.Second
}

// Run ticks until ctx is cancelled.
func (s *Sweeper) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		s.Tick()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick marks every service whose heartbeat has gone stale as unknown.
func (s *Sweeper) Tick() {
	s.DB.Model(&models.ServiceLiveness{}).
		Where("state != ? AND last_heartbeat < ?", models.LivenessUnknown, s.now().Add(-s.stale())).
		Update("state", models.LivenessUnknown)
}
