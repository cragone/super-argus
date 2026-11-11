package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// journalEntry is the subset of `journalctl -o json` fields the agent uses.
type journalEntry struct {
	Message    any    `json:"MESSAGE"`
	Unit       string `json:"_SYSTEMD_UNIT"`
	Identifier string `json:"SYSLOG_IDENTIFIER"`
	Host       string `json:"_HOSTNAME"`
	Realtime   string `json:"__REALTIME_TIMESTAMP"` // microseconds since epoch
	Cursor     string `json:"__CURSOR"`
}

// journalService names the service for a journal entry: the systemd unit
// without its suffix, falling back to the syslog identifier. Login sessions
// share one name so their changing ids do not create a service each.
func journalService(e journalEntry) string {
	name := e.Unit
	if name == "" {
		name = e.Identifier
	}
	if name == "" {
		return "system"
	}
	if strings.HasPrefix(name, "session-") && strings.HasSuffix(name, ".scope") {
		return "user-sessions"
	}
	return strings.TrimSuffix(name, ".service")
}

// journalMessage returns the text of MESSAGE. journald encodes non-UTF-8
// messages as an array of byte values.
func journalMessage(m any) (string, bool) {
	switch v := m.(type) {
	case string:
		return v, true
	case []any:
		b := make([]byte, 0, len(v))
		for _, x := range v {
			f, ok := x.(float64)
			if !ok {
				return "", false
			}
			b = append(b, byte(f))
		}
		return string(b), true
	}
	return "", false
}

func journalTime(s string) time.Time {
	us, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Now()
	}
	return time.UnixMicro(us)
}

// parseJournalLine converts one JSON line to labels, timestamp, text and cursor.
func parseJournalLine(raw []byte) (labels map[string]string, ts time.Time, line, cursor string, ok bool) {
	var e journalEntry
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, time.Time{}, "", "", false
	}
	line, ok = journalMessage(e.Message)
	if !ok || line == "" {
		return nil, time.Time{}, "", e.Cursor, false
	}
	labels = map[string]string{"service": journalService(e), "source": "journald"}
	if e.Unit != "" {
		labels["unit"] = e.Unit
	}
	if e.Host != "" {
		labels["host"] = e.Host
	}
	return labels, journalTime(e.Realtime), line, e.Cursor, true
}

// runJournal follows journald until ctx ends, restarting journalctl if it exits.
func runJournal(ctx context.Context, hub *Hub, st *State, units []string) {
	backoff := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		if err := followJournal(ctx, hub, st, units); err != nil && ctx.Err() == nil {
			log.Printf("argus-agent: journalctl: %v", err)
		}
		if time.Since(started) > time.Minute {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// normalizeUnit appends ".service" to bare unit names ("nginx" -> "nginx.service").
func normalizeUnit(u string) string {
	if strings.Contains(u, ".") {
		return u
	}
	return u + ".service"
}

// journalArgs builds the journalctl arguments, matching only the given units.
// journalctl ORs repeated -u flags.
func journalArgs(cursor string, units []string) []string {
	args := []string{"-o", "json", "--no-pager", "-f"}
	for _, u := range units {
		args = append(args, "-u", u)
	}
	if cursor != "" {
		args = append(args, "--after-cursor="+cursor)
	} else {
		args = append(args, "-n", "0") // first run: only new lines, not the whole journal
	}
	return args
}

func followJournal(ctx context.Context, hub *Hub, st *State, units []string) error {
	want := make(map[string]bool, len(units))
	for _, u := range units {
		want[u] = true
	}
	args := journalArgs(st.GetCursor(), units)
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		labels, ts, line, cursor, ok := parseJournalLine(sc.Bytes())
		if cursor != "" {
			st.SetCursor(cursor)
		}
		if ok {
			hub.Send(labels, ts, line)
		}
	}
	_ = cmd.Wait()
	return sc.Err()
}
