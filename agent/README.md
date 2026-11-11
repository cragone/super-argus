# argus-agent

Run it on a host or node and everything running there shows up in Argus under **Services**. Pods are discovered automatically; systemd units are collected only when listed in `ARGUS_JOURNAL_UNITS`.

| Mode | Source | Service name |
|------|--------|--------------|
| `journal` | systemd journal (`journalctl -f`) | the unit, e.g. `nginx.service` becomes `nginx` |
| `kubernetes` | `/var/log/pods` container logs | the owning workload, e.g. pod `api-7d9f8b6c5d-x2k4p` becomes `api` |

`ARGUS_AGENT_MODE=auto` (the default) uses `kubernetes` if `/var/log/pods` exists, else `journal`.

Pod and workload names come from the log file path, not the Kubernetes API, so the agent needs no RBAC to ship logs. Labels attached: `service`, `namespace`, `pod`, `container` (Kubernetes); `service`, `unit`, `host` (journal); plus `source` and anything in `ARGUS_LABELS`.

## Liveness

Separately from logs, the agent reports whether each service is actually running — systemd's own `ActiveState`/`SubState` (journal mode) or the pod's `Ready` condition from the Kubernetes API (kubernetes mode) — not just "logged recently". This shows up as the colored `alive`/`down`/`unknown` badge on the Services page and in `argus tui`, alongside (not instead of) the existing log-activity badge.

- **journal mode**: polls systemd over D-Bus every 10s for the units named in `ARGUS_JOURNAL_UNITS` (the same ones being tailed, nothing extra), reusing the same unit → service name mapping as the log tailer. No RBAC or extra config; reading unit state is allowed to any user by systemd's default policy.
- **kubernetes mode**: watches this node's pods via the Kubernetes API (needs the RBAC and `ARGUS_NODE_NAME` in `deploy/daemonset.yaml` — `get`/`list`/`watch` on `pods`, cluster-scoped since RBAC has no per-node granularity, though each agent only queries its own node). A service counts as alive if any of its replicas on the node is Ready, so one crash-looping replica does not flip a healthy service to "down". Without the RBAC, this collector logs once and skips itself; logs still ship normally.
- Both report a heartbeat every ~10-15s even when nothing changed, batched and flushed to `POST /api/v1/ingest/liveness` every 15s, so Argus can tell "confirmed down" apart from "this agent stopped reporting" (which surfaces as `unknown` after about 45s of silence).

## Configuration

| Variable | Default | Meaning |
|----------|---------|---------|
| `ARGUS_API_KEY` | required | ingest key (`POST /api/v1/api-keys` as admin) |
| `ARGUS_URL` | `http://localhost:8080` | Argus base URL |
| `ARGUS_AGENT_MODE` | `auto` | `auto`, `journal` or `kubernetes` |
| `ARGUS_JOURNAL_UNITS` | (required in journal mode) | comma-separated systemd units to collect, e.g. `nginx,cups.service`; other units, the kernel and scopes are not read |
| `ARGUS_LABELS` | | extra labels for every stream: `env=prod,cluster=a` |
| `ARGUS_EXCLUDE_NAMESPACES` | | comma list of namespaces to skip (Kubernetes) |
| `ARGUS_NODE_NAME` | | this node's name (Kubernetes liveness only; set via `fieldRef: spec.nodeName` in `deploy/daemonset.yaml`). Without it, pod liveness is skipped but logs still ship. |
| `ARGUS_PODS_DIR` | `/var/log/pods` | pod log root |
| `ARGUS_STATE_DIR` | `/var/lib/argus-agent` | where read positions are kept |
| `ARGUS_ACTIONS_ENABLED` | | `true` lets Argus admins restart things through this agent (see Restart actions) |
| `ARGUS_ACTION_UNITS` | | journal mode: units the agent may restart, e.g. `nginx,cups.service`; empty means none |
| `ARGUS_ACTION_NAMESPACES` | | kubernetes mode: namespaces the agent may restart workloads in; empty means none |
| `ARGUS_ACTION_HOST` | `os.Hostname()` | the `host` label this agent answers to; must match the journald hostname |

## Install

systemd host:

```sh
cd agent && GOOS=linux go build -o argus-agent . && sudo install argus-agent /usr/local/bin/
printf 'ARGUS_URL=http://argus:8080\nARGUS_API_KEY=argus_xxx\n' | sudo tee /etc/argus-agent.env
sudo cp deploy/argus-agent.service /etc/systemd/system/ && sudo systemctl enable --now argus-agent
```

Kubernetes: build `Dockerfile` from the repo root, push it, then follow the header of `deploy/daemonset.yaml`.

## Restart actions

Argus admins can restart a systemd unit or a pod from the UI or `POST /api/v1/services/{name}/restart`. The server only queues the request; the agent that owns the target polls for it (outbound only, no listener), checks its **own** allowlist, runs it and reports the result. Every request, refused ones included, is kept in `GET /api/v1/actions`.

- **systemd:** needs `ARGUS_ACTIONS_ENABLED=true`, `ARGUS_ACTION_UNITS`, and a polkit rule so the unprivileged agent can restart exactly those units: `deploy/argus-agent.rules` (the NixOS module's `services.argus.agent.actions` does both). Without the rule the restart fails with an authorization error.
- **Kubernetes:** needs `ARGUS_ACTIONS_ENABLED=true`, `ARGUS_ACTION_NAMESPACES`, and a RoleBinding per allowed namespace (example at the end of `deploy/daemonset.yaml`). A pod owned by a Deployment, StatefulSet or DaemonSet gets a rolling restart (the `kubectl rollout restart` annotation); a pod with no owner is deleted. Pods owned by anything else (Jobs, operators) are refused.
- Only the agent on the pod's node acts on a pod, and only the agent on the matching host acts on a unit.

## Behavior to know

- On first start the agent follows files and the journal from now on; it does not replay history. After that it resumes from the saved position, so a restart neither loses nor duplicates lines.
- New pod log files are found within about 5 seconds and read from their start.
- Kubernetes rotates a container's log by renaming `0.log`; the last lines written to the old file just before a rotation can be missed.
- Every distinct label set is one stream with its own queue. The agent stops creating new ones after 500, and Argus enforces `MaxStreamsPerKey` too. Keep `ARGUS_LABELS` low-cardinality.
- If Argus is down, lines are retried a few times and then dropped rather than buffered without limit.
- Not yet verified against a real cluster or a real journal; the unit tests cover parsing, discovery and the offset logic with a fake log directory.
