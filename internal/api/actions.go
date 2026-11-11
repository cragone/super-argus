package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cragone/Super-Argus/internal/auth"
	"github.com/cragone/Super-Argus/internal/ingest"
	"github.com/cragone/Super-Argus/internal/models"
	"github.com/go-chi/chi/v5"
)

const (
	// actionTTL is how long a pending action waits for an agent before expiring.
	actionTTL = 2 * time.Minute
	// restartCooldown stops a UI double-click or a loop from restarting a target repeatedly.
	restartCooldown = 30 * time.Second
	// pendingPollWait is how long an agent's pending poll is held open.
	pendingPollWait = 25 * time.Second
)

type actionView struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Service     string            `json:"service"`
	Target      map[string]string `json:"target"`
	Status      string            `json:"status"`
	RequestedBy string            `json:"requested_by"`
	RequestedAt time.Time         `json:"requested_at"`
	FinishedAt  *time.Time        `json:"finished_at,omitempty"`
	Result      string            `json:"result,omitempty"`
}

func viewAction(a models.Action) actionView {
	t := map[string]string{}
	_ = json.Unmarshal([]byte(a.Target), &t)
	return actionView{ID: a.ID, Kind: a.Kind, Service: a.Service, Target: t, Status: a.Status,
		RequestedBy: a.RequestedBy, RequestedAt: a.RequestedAt, FinishedAt: a.FinishedAt, Result: a.Result}
}

type restartInput struct {
	// Labels are the labels of the stream being viewed (source, host, unit,
	// namespace, pod...). The kind and target are derived from them server-side.
	Labels map[string]string `json:"labels"`
}

// restartTarget derives the action kind and the minimal target labels from stream labels.
func restartTarget(service string, labels map[string]string) (kind string, target map[string]string, msg string) {
	switch labels["source"] {
	case "journald":
		unit := labels["unit"]
		if unit == "" {
			unit = service + ".service"
		}
		if labels["host"] == "" {
			return "", nil, "labels.host is required to restart a systemd unit"
		}
		return models.ActionSystemdRestart, map[string]string{"host": labels["host"], "unit": unit}, ""
	case "kubernetes":
		if labels["namespace"] == "" || labels["pod"] == "" {
			return "", nil, "labels.namespace and labels.pod are required to restart a pod"
		}
		return models.ActionK8sRestart, map[string]string{"namespace": labels["namespace"], "pod": labels["pod"]}, ""
	}
	return "", nil, "labels.source must be journald or kubernetes"
}

// expireStale marks pending actions nobody claimed in time as expired.
func (s *Server) expireStale() {
	now := time.Now().UTC()
	s.DB.Model(&models.Action{}).Where("status = ? AND requested_at < ?", models.ActionPending, now.Add(-actionTTL)).
		Updates(map[string]any{"status": models.ActionExpired, "finished_at": now, "result": "no agent claimed the action in time"})
}

// requestRestart queues a restart for the agent that owns the target (admin only).
func (s *Server) requestRestart(w http.ResponseWriter, r *http.Request) {
	var in restartInput
	if !decode(w, r, &in) {
		return
	}
	service := chi.URLParam(r, "name")
	kind, target, msg := restartTarget(service, in.Labels)
	if msg != "" {
		writeError(w, http.StatusBadRequest, "bad_request", msg)
		return
	}
	tj, _ := json.Marshal(target)
	s.expireStale()

	var recent int64
	s.DB.Model(&models.Action{}).
		Where("kind = ? AND target = ? AND (status IN ? OR requested_at > ?)", kind, string(tj),
			[]string{models.ActionPending, models.ActionRunning}, time.Now().UTC().Add(-restartCooldown)).
		Count(&recent)
	if recent > 0 {
		writeError(w, http.StatusConflict, "conflict", "a restart of this target is already queued or ran moments ago")
		return
	}
	p, _ := PrincipalFrom(r.Context())
	a := models.Action{Kind: kind, Service: service, Target: string(tj), Status: models.ActionPending,
		RequestedBy: p.UserID, RequestedAt: time.Now().UTC()}
	if err := s.DB.Create(&a).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not queue action")
		return
	}
	writeJSON(w, http.StatusAccepted, viewAction(a))
}

func (s *Server) listActions(w http.ResponseWriter, _ *http.Request) {
	s.expireStale()
	var as []models.Action
	if err := s.DB.Order("requested_at DESC").Limit(100).Find(&as).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "query failed")
		return
	}
	out := make([]actionView, len(as))
	for i, a := range as {
		out[i] = viewAction(a)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) getAction(w http.ResponseWriter, r *http.Request) {
	s.expireStale()
	var a models.Action
	if !s.find(w, &a, chi.URLParam(r, "id")) {
		return
	}
	writeJSON(w, http.StatusOK, viewAction(a))
}

// agentKey authenticates an agent by its ingest API key.
func (s *Server) agentKey(w http.ResponseWriter, r *http.Request) (*models.APIKey, bool) {
	k := r.Header.Get(ingest.KeyHeader)
	if k == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing API key")
		return nil, false
	}
	key, err := auth.LookupAPIKey(s.DB, k)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or revoked API key")
		return nil, false
	}
	return key, true
}

// agentPending long-polls for actions an agent may be able to run. systemd
// actions are filtered to the agent's host; k8s actions are returned to every
// agent, which claims only the ones whose pod it can see locally.
func (s *Server) agentPending(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.agentKey(w, r); !ok {
		return
	}
	host := r.URL.Query().Get("host")
	deadline := time.NewTimer(pendingPollWait)
	defer deadline.Stop()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		s.expireStale()
		var as []models.Action
		if err := s.DB.Where("status = ?", models.ActionPending).Order("requested_at").Find(&as).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "query failed")
			return
		}
		var out []actionView
		for _, a := range as {
			v := viewAction(a)
			if a.Kind == models.ActionSystemdRestart && (host == "" || v.Target["host"] != host) {
				continue
			}
			out = append(out, v)
		}
		if len(out) > 0 {
			writeJSON(w, http.StatusOK, map[string]any{"items": out})
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			writeJSON(w, http.StatusOK, map[string]any{"items": []actionView{}})
			return
		case <-tick.C:
		}
	}
}

// agentClaim atomically moves one pending action to running for the calling agent.
func (s *Server) agentClaim(w http.ResponseWriter, r *http.Request) {
	key, ok := s.agentKey(w, r)
	if !ok {
		return
	}
	now := time.Now().UTC()
	res := s.DB.Model(&models.Action{}).Where("id = ? AND status = ?", chi.URLParam(r, "id"), models.ActionPending).
		Updates(map[string]any{"status": models.ActionRunning, "claimed_by": key.ID, "claimed_at": now})
	if res.Error != nil {
		writeError(w, http.StatusInternalServerError, "internal", "claim failed")
		return
	}
	if res.RowsAffected == 0 {
		writeError(w, http.StatusConflict, "conflict", "action is not pending")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type actionResult struct {
	OK     bool   `json:"ok"`
	Result string `json:"result"`
}

// agentResult records the outcome of a claimed action; only the claimer may report.
func (s *Server) agentResult(w http.ResponseWriter, r *http.Request) {
	key, ok := s.agentKey(w, r)
	if !ok {
		return
	}
	var in actionResult
	if !decode(w, r, &in) {
		return
	}
	status := models.ActionSucceeded
	if !in.OK {
		status = models.ActionFailed
	}
	if len(in.Result) > 2000 {
		in.Result = in.Result[:2000]
	}
	res := s.DB.Model(&models.Action{}).
		Where("id = ? AND status = ? AND claimed_by = ?", chi.URLParam(r, "id"), models.ActionRunning, key.ID).
		Updates(map[string]any{"status": status, "finished_at": time.Now().UTC(), "result": strings.TrimSpace(in.Result)})
	if res.Error != nil {
		writeError(w, http.StatusInternalServerError, "internal", "update failed")
		return
	}
	if res.RowsAffected == 0 {
		writeError(w, http.StatusConflict, "conflict", "action is not running under this key")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// targetWindow is how recently a stream must have logged to count as a live restart target.
const targetWindow = 15 * time.Minute

type targetView struct {
	Kind     string            `json:"kind"`
	Target   map[string]string `json:"target"`
	Labels   map[string]string `json:"labels"`
	LastSeen time.Time         `json:"last_seen"`
}

// restartTargets lists what could be restarted for a service, derived from its
// recently active streams (admin only). Labels is what to send back to the
// restart endpoint.
func (s *Server) restartTargets(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var streams []models.Stream
	err := s.DB.Model(&models.Stream{}).
		Joins("JOIN stream_labels ON stream_labels.stream_id = streams.id AND stream_labels.key = ? AND stream_labels.value = ?", "service", name).
		Where("streams.last_seen >= ?", time.Now().UTC().Add(-targetWindow)).
		Order("streams.last_seen DESC").Find(&streams).Error
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "query failed")
		return
	}
	out := []targetView{}
	seen := map[string]bool{}
	for _, st := range streams {
		labels := map[string]string{}
		if json.Unmarshal([]byte(st.Labels), &labels) != nil {
			continue
		}
		kind, target, msg := restartTarget(name, labels)
		if msg != "" {
			continue
		}
		tj, _ := json.Marshal(target)
		if key := kind + string(tj); seen[key] {
			continue
		} else {
			seen[key] = true
		}
		out = append(out, targetView{Kind: kind, Target: target, Labels: restartLabels(labels["source"], target), LastSeen: st.LastSeen})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// restartLabels is the minimal label set restartTarget accepts for a target.
func restartLabels(source string, target map[string]string) map[string]string {
	l := map[string]string{"source": source}
	for k, v := range target {
		l[k] = v
	}
	return l
}
