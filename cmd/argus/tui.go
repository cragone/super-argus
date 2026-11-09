package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// tuiConfig is a process-compose config generated on the fly: a read-only
// monitoring view that tails the argus/argus-agent systemd units' logs side
// by side. There's no supervising process-compose instance in production
// (systemd runs the binaries directly, per nix/module.nix), so this isn't an
// "attach" to anything already running -- it's its own short-lived
// process-compose session just for the dashboard. Killing/restarting a
// process in the TUI only restarts that log tail, not the real
// systemd-managed service; use systemctl for that.
const tuiConfig = `version: "0.5"
processes:
  argus:
    command: "journalctl -u argus.service -f -n 200 -o short-iso --no-hostname"
    availability:
      restart: "no"
  argus-agent:
    command: "journalctl -u argus-agent.service -f -n 200 -o short-iso --no-hostname"
    availability:
      restart: "no"
`

// runTUI opens the process-compose dashboard. Extra args are passed through
// to `process-compose up` (e.g. `argus tui --tui=false` to run headless).
func runTUI(args []string, log *slog.Logger) int {
	pc, err := exec.LookPath("process-compose")
	if err != nil {
		fmt.Fprintln(os.Stderr, "argus tui: process-compose not found on PATH (nix: pkgs.process-compose)")
		return 1
	}

	f, err := os.CreateTemp("", "argus-tui-*.yaml")
	if err != nil {
		log.Error("tui: create temp config", "err", err)
		return 1
	}
	defer os.Remove(f.Name())
	_, werr := f.WriteString(tuiConfig)
	cerr := f.Close()
	if werr != nil {
		log.Error("tui: write temp config", "err", werr)
		return 1
	}
	if cerr != nil {
		log.Error("tui: close temp config", "err", cerr)
		return 1
	}

	cmd := exec.Command(pc, append([]string{"up", "-f", f.Name()}, args...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return exitErr.ExitCode()
		}
		log.Error("tui: run process-compose", "err", err)
		return 1
	}
	return 0
}
