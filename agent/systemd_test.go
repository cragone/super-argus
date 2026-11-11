package main

import (
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
)

func TestSystemdServicePing(t *testing.T) {
	cases := []struct {
		name  string
		unit  dbus.UnitStatus
		want  string
		alive bool
		ok    bool
	}{
		{"service, running", dbus.UnitStatus{Name: "nginx.service", ActiveState: "active", SubState: "running"}, "nginx", true, true},
		{"service, active but not running (oneshot)", dbus.UnitStatus{Name: "backup.service", ActiveState: "active", SubState: "exited"}, "backup", false, true},
		{"service, failed", dbus.UnitStatus{Name: "nginx.service", ActiveState: "failed", SubState: "failed"}, "nginx", false, true},
		{"socket unit is skipped", dbus.UnitStatus{Name: "docker.socket", ActiveState: "active", SubState: "running"}, "", false, false},
		{"timer unit is skipped", dbus.UnitStatus{Name: "backup.timer", ActiveState: "active", SubState: "waiting"}, "", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name, alive, ok := systemdServicePing(c.unit)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if !ok {
				return
			}
			if name != c.want {
				t.Errorf("name = %q, want %q", name, c.want)
			}
			if alive != c.alive {
				t.Errorf("alive = %v, want %v", alive, c.alive)
			}
		})
	}
}
