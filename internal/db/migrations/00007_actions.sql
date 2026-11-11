-- +goose Up
-- actions is both the command queue the agents poll and the audit log of who
-- asked for what; rows are never deleted by the app.
CREATE TABLE actions (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('systemd_restart','k8s_restart')),
    service TEXT NOT NULL,
    target TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','succeeded','failed','expired')),
    requested_by TEXT NOT NULL,
    requested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    claimed_by TEXT NOT NULL DEFAULT '',
    claimed_at TIMESTAMP,
    finished_at TIMESTAMP,
    result TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_actions_status ON actions (status, requested_at);
CREATE INDEX idx_actions_target ON actions (kind, service, requested_at);

-- +goose Down
DROP TABLE actions;
