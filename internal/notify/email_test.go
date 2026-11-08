package notify

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cragone/Super-Argus/internal/models"
	"github.com/stretchr/testify/require"
)

// fakeSMTP is a minimal SMTP server good enough for net/smtp.SendMail: EHLO,
// optional AUTH PLAIN, MAIL FROM/RCPT TO/DATA, QUIT. It does not advertise
// STARTTLS, so SendMail stays in plaintext.
type fakeSMTP struct {
	ln net.Listener

	mu       sync.Mutex
	bodies   []string
	authUser string
	failCode int // if nonzero, every DATA command gets this SMTP reply code instead of success
}

func startFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	f := &fakeSMTP{ln: ln}
	go f.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return f
}

func (f *fakeSMTP) addr() string { return f.ln.Addr().String() }

func (f *fakeSMTP) serve() {
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

func (f *fakeSMTP) handle(conn net.Conn) {
	defer conn.Close()
	tp := textproto.NewConn(conn)
	tp.PrintfLine("220 fake.smtp ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			tp.PrintfLine("250-fake.smtp")
			tp.PrintfLine("250 AUTH PLAIN")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			parts := strings.SplitN(line, " ", 3)
			if len(parts) == 3 {
				raw, _ := base64.StdEncoding.DecodeString(parts[2])
				seg := strings.Split(string(raw), "\x00")
				if len(seg) == 3 {
					f.mu.Lock()
					f.authUser = seg[1]
					f.mu.Unlock()
				}
			}
			tp.PrintfLine("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM"):
			tp.PrintfLine("250 ok")
		case strings.HasPrefix(upper, "RCPT TO"):
			tp.PrintfLine("250 ok")
		case upper == "DATA":
			f.mu.Lock()
			code := f.failCode
			f.mu.Unlock()
			if code != 0 {
				tp.PrintfLine("%d rejected", code)
				continue
			}
			tp.PrintfLine("354 go ahead")
			r := bufio.NewReader(tp.R)
			var body strings.Builder
			for {
				l, err := readDotLine(r)
				if err != nil || l == "." {
					break
				}
				body.WriteString(l)
				body.WriteString("\n")
			}
			f.mu.Lock()
			f.bodies = append(f.bodies, body.String())
			f.mu.Unlock()
			tp.PrintfLine("250 ok: queued")
		case upper == "QUIT":
			tp.PrintfLine("221 bye")
			return
		default:
			tp.PrintfLine("500 unrecognized command")
		}
	}
}

func readDotLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func emailChannel(t *testing.T, s *Sealer, host string, port int, password string) models.NotificationChannel {
	cfg := EmailConfig{SMTPHost: host, SMTPPort: port, From: "alerts@argus.local", To: []string{"oncall@example.com"}}
	if password != "" {
		cfg.Username = "argus"
		enc, err := s.Seal(password)
		require.NoError(t, err)
		cfg.PasswordEnc = enc
	}
	b, _ := json.Marshal(cfg)
	return models.NotificationChannel{Name: "c", Type: "email", Config: string(b)}
}

func TestEmailSendsWithAuth(t *testing.T) {
	srv := startFakeSMTP(t)
	host, portStr, _ := net.SplitHostPort(srv.addr())
	port, _ := strconv.Atoi(portStr)

	s := NewSealer("k")
	e := &Email{Sealer: s, Sleep: func(time.Duration) {}}
	ch := emailChannel(t, s, host, port, "hunter2")
	require.NoError(t, e.Send(t.Context(), ch, Payload{Status: "firing", RuleName: "high-error-rate"}))

	srv.mu.Lock()
	defer srv.mu.Unlock()
	require.Equal(t, "argus", srv.authUser)
	require.Len(t, srv.bodies, 1)
	require.Contains(t, srv.bodies[0], "Rule: high-error-rate")
}

func TestEmailRetriesOn4xxAndGivesUpOn5xx(t *testing.T) {
	srv := startFakeSMTP(t)
	host, portStr, _ := net.SplitHostPort(srv.addr())
	port, _ := strconv.Atoi(portStr)

	s := NewSealer("k")
	var sleeps []time.Duration
	e := &Email{Sealer: s, Sleep: func(d time.Duration) { sleeps = append(sleeps, d) }}
	ch := emailChannel(t, s, host, port, "")

	srv.mu.Lock()
	srv.failCode = 450 // transient, should retry
	srv.mu.Unlock()
	err := e.Send(t.Context(), ch, Payload{Status: "firing"})
	require.Error(t, err)
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second}, sleeps)

	sleeps = nil
	srv.mu.Lock()
	srv.failCode = 550 // permanent, no retry
	srv.mu.Unlock()
	err = e.Send(t.Context(), ch, Payload{Status: "firing"})
	require.Error(t, err)
	require.Empty(t, sleeps, "permanent SMTP failure is not retried")
}
