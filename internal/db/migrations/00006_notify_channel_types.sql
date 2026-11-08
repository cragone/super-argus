-- +goose Up
-- SQLite can't ALTER a CHECK constraint in place; rebuild the table. This
-- form (create new, copy, drop, rename) is plain SQL and works unchanged on
-- Postgres too. No other table has a foreign key into notification_channels
-- (notify_channel_ids on alert_rules is a JSON column, not an FK), so this
-- table can be rebuilt on its own.
CREATE TABLE notification_channels_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('webhook', 'email', 'sms')),
    config TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO notification_channels_new (id, name, type, config, created_at, updated_at)
    SELECT id, name, type, config, created_at, updated_at FROM notification_channels;
DROP TABLE notification_channels;
ALTER TABLE notification_channels_new RENAME TO notification_channels;

-- +goose Down
CREATE TABLE notification_channels_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('webhook')),
    config TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO notification_channels_old (id, name, type, config, created_at, updated_at)
    SELECT id, name, type, config, created_at, updated_at FROM notification_channels;
DROP TABLE notification_channels;
ALTER TABLE notification_channels_old RENAME TO notification_channels;
