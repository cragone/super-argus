package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"
)

// restartClient talks to a running Argus server as an admin user. It is the
// CLI counterpart of the UI's Restart button: the server queues the request and
// the agent that owns the target performs it.
type restartClient struct {
	base string
	http *http.Client
	in   io.Reader
	out  io.Writer
	// poll is how often the action is checked; wait is how long before giving up.
	poll, wait time.Duration
	token      string
}

type restartTarget struct {
	Kind   string            `json:"kind"`
	Target map[string]string `json:"target"`
	Labels map[string]string `json:"labels"`
}

func (t restartTarget) String() string {
	if t.Kind == "systemd_restart" {
		return t.Target["unit"] + " on " + t.Target["host"]
	}
	return "pod " + t.Target["namespace"] + "/" + t.Target["pod"]
}

type restartAction struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
}

// do sends a JSON request and decodes a JSON response; non-2xx becomes an error
// carrying the server's message.
func (c *restartClient) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+"/api/v1"+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		var e struct {
			Error struct{ Message string } `json:"error"`
		}
		if json.Unmarshal(b, &e) == nil && e.Error.Message != "" {
			return fmt.Errorf("%s (HTTP %d)", e.Error.Message, resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(b, out)
	}
	return nil
}

func (c *restartClient) login(ctx context.Context, email, password string) error {
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do(ctx, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": password}, &tok); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	c.token = tok.AccessToken
	return nil
}

// restart picks a target, confirms (unless yes), queues the restart and waits
// for the agent's result. It returns an error unless the restart succeeded.
func (c *restartClient) restart(ctx context.Context, service string, pick int, yes bool) error {
	var targets struct {
		Items []restartTarget `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, "/services/"+url.PathEscape(service)+"/restart-targets", nil, &targets); err != nil {
		return err
	}
	items := targets.Items
	if len(items) == 0 {
		return fmt.Errorf("%q has no restartable target (no recently active journald or kubernetes stream)", service)
	}
	if pick == 0 && len(items) > 1 {
		fmt.Fprintf(c.out, "%q has %d targets; choose one with -target N:\n", service, len(items))
		for i, t := range items {
			fmt.Fprintf(c.out, "  %d. %s\n", i+1, t)
		}
		return errors.New("ambiguous target")
	}
	if pick == 0 {
		pick = 1
	}
	if pick < 1 || pick > len(items) {
		return fmt.Errorf("-target %d out of range (1-%d)", pick, len(items))
	}
	t := items[pick-1]

	if !yes {
		fmt.Fprintf(c.out, "Restart %s? [y/N] ", t)
		line, _ := bufio.NewReader(c.in).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
			return errors.New("aborted")
		}
	}

	var act restartAction
	if err := c.do(ctx, http.MethodPost, "/services/"+url.PathEscape(service)+"/restart", map[string]any{"labels": t.Labels}, &act); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "queued %s, waiting for the agent...\n", act.ID)

	deadline := time.Now().Add(c.wait)
	for {
		switch act.Status {
		case "succeeded":
			fmt.Fprintf(c.out, "done: %s\n", act.Result)
			return nil
		case "failed", "expired":
			if act.Result == "" {
				act.Result = act.Status
			}
			return fmt.Errorf("%s: %s", act.Status, act.Result)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for the agent (action %s is still %s)", act.ID, act.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.poll):
		}
		if err := c.do(ctx, http.MethodGet, "/actions/"+url.PathEscape(act.ID), nil, &act); err != nil {
			return err
		}
	}
}

func runRestart(args []string, log *slog.Logger) int {
	fs := flag.NewFlagSet("restart", flag.ContinueOnError)
	srv := fs.String("url", envOr("ARGUS_URL", "http://localhost:8080"), "Argus server URL")
	pick := fs.Int("target", 0, "which target to restart when the service has several (1-based)")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	email, pw := os.Getenv("ARGUS_EMAIL"), os.Getenv("ARGUS_PASSWORD")
	if email == "" || pw == "" {
		fmt.Fprintln(os.Stderr, "argus restart: set ARGUS_EMAIL and ARGUS_PASSWORD (an admin account)")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	c := &restartClient{base: strings.TrimRight(*srv, "/"), http: &http.Client{Timeout: 30 * time.Second},
		in: os.Stdin, out: os.Stdout, poll: 2 * time.Second, wait: 3 * time.Minute}
	if err := c.login(ctx, email, pw); err != nil {
		log.Error("restart", "err", err)
		return 1
	}
	if err := c.restart(ctx, fs.Arg(0), *pick, *yes); err != nil {
		fmt.Fprintln(os.Stderr, "argus restart:", err)
		return 1
	}
	return 0
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
