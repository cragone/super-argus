package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

// runSystemdLiveness polls systemd over D-Bus for every unit's ActiveState
// and SubState, reusing journalService() so a unit's liveness lines up with
// the service name its journal lines are already filed under. Nothing is
// configured per unit, matching the rest of the agent.
func runSystemdLiveness(ctx context.Context, report *LivenessReporter, interval time.Duration) {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		log.Printf("argus-agent: systemd liveness: connect to dbus: %v", err)
		return
	}
	defer conn.Close()

	t := time.NewTicker(interval)
	defer t.Stop()
	for ctx.Err() == nil {
		pollSystemdUnits(ctx, conn, report)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// pollSystemdUnits reports liveness for every ".service" unit systemd knows
// about. Other unit types (sockets, timers, mounts, targets) are skipped —
// they are not what "service" means on the Services page.
func pollSystemdUnits(ctx context.Context, conn *dbus.Conn, report *LivenessReporter) {
	units, err := conn.ListUnitsContext(ctx)
	if err != nil {
		log.Printf("argus-agent: systemd liveness: list units: %v", err)
		return
	}
	for _, u := range units {
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
