package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

// runSystemdLiveness polls systemd over D-Bus for the ActiveState/SubState of
// exactly the units named in ARGUS_JOURNAL_UNITS, reusing journalService() so
// a unit's liveness lines up with the service name its journal lines are
// already filed under. It deliberately does not report on every unit on the
// host: journal mode only collects the units it is told to (see
// agent/journal.go), and liveness follows the same allowlist rather than
// surfacing services nothing is logging.
func runSystemdLiveness(ctx context.Context, report *LivenessReporter, interval time.Duration, units []string) {
	allowed := make(map[string]bool, len(units))
	for _, u := range units {
		allowed[u] = true
	}
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		log.Printf("argus-agent: systemd liveness: connect to dbus: %v", err)
		return
	}
	defer conn.Close()

	t := time.NewTicker(interval)
	defer t.Stop()
	for ctx.Err() == nil {
		pollSystemdUnits(ctx, conn, report, allowed)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// pollSystemdUnits reports liveness for every unit in allowed. Units not in
// the allowlist are skipped, same as journal mode's own log collection.
func pollSystemdUnits(ctx context.Context, conn *dbus.Conn, report *LivenessReporter, allowed map[string]bool) {
	units, err := conn.ListUnitsContext(ctx)
	if err != nil {
		log.Printf("argus-agent: systemd liveness: list units: %v", err)
		return
	}
	for _, u := range units {
		if !allowed[u.Name] {
			continue
		}
		if name, alive, ok := systemdServicePing(u); ok {
			report.Report(name, "systemd", alive, u.SubState)
		}
	}
}

// systemdServicePing reduces one systemd unit to a liveness report: ok is
// false for anything that is not a ".service" unit. name reuses
// journalService() so it lines up with the service's log stream.
func systemdServicePing(u dbus.UnitStatus) (name string, alive bool, ok bool) {
	if !strings.HasSuffix(u.Name, ".service") {
		return "", false, false
	}
	name = journalService(journalEntry{Unit: u.Name})
	alive = u.ActiveState == "active" && u.SubState == "running"
	return name, alive, true
}
