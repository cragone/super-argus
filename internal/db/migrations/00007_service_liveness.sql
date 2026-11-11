-- +goose Up
CREATE TABLE service_liveness (
    service TEXT PRIMARY KEY,
    source TEXT NOT NULL CHECK (source IN ('systemd', 'kubernetes')),
    state TEXT NOT NULL CHECK (state IN ('alive', 'down', 'unknown')),
    detail TEXT NOT NULL DEFAULT '',
    last_heartbeat TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE service_liveness;
