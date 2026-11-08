package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/cragone/Super-Argus/internal/models"
)

// EmailConfig is the JSON stored in NotificationChannel.Config for type
// "email". PasswordEnc is the Sealer-encrypted SMTP password (optional; omit
// Username/PasswordEnc entirely for a server that doesn't require auth).
type EmailConfig struct {
	SMTPHost    string   `json:"smtp_host"`
	SMTPPort    int      `json:"smtp_port"`
	Username    string   `json:"username,omitempty"`
	PasswordEnc string   `json:"password_enc,omitempty"`
	From        string   `json:"from"`
	To          []string `json:"to"`
}

// smtpSender abstracts net/smtp.SendMail so tests can inject a fake without a
// real SMTP server.
type smtpSender interface {
	SendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error
}

type stdlibSMTP struct{}

func (stdlibSMTP) SendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, auth, from, to, msg)
}

// Email delivers payloads over SMTP with retry and backoff. SMTP 4xx replies
// (transient negative completion) are retried; SMTP 5xx (permanent) and any
// other non-transient failure fail immediately. This is the inverse of HTTP's
// convention, where 5xx is retried and 4xx is not.
type Email struct {
	Sealer   *Sealer
	Attempts int                 // default 3
	Backoff  time.Duration       // first delay, doubled each retry; default 1s
	Sleep    func(time.Duration) // injectable for tests
	Dial     smtpSender          // injectable for tests; defaults to stdlibSMTP{}
}

// Send implements Notifier.
func (e *Email) Send(ctx context.Context, ch models.NotificationChannel, p Payload) error {
	var cfg EmailConfig
	if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil || cfg.SMTPHost == "" || cfg.From == "" || len(cfg.To) == 0 {
		return errors.New("channel has no valid email config")
	}
	var auth smtp.Auth
	if cfg.Username != "" {
		var password string
		if cfg.PasswordEnc != "" && e.Sealer != nil {
			pw, err := e.Sealer.Open(cfg.PasswordEnc)
			if err != nil {
				return errors.New("cannot decrypt channel secret")
			}
			password = pw
		}
		auth = smtp.PlainAuth("", cfg.Username, password, cfg.SMTPHost)
	}
	msg := buildEmailMessage(cfg.From, cfg.To, p)

	attempts, backoff, sleep := e.Attempts, e.Backoff, e.Sleep
	if attempts <= 0 {
		attempts = 3
	}
	if backoff <= 0 {
		backoff = time.Second
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	sender := e.Dial
	if sender == nil {
		sender = stdlibSMTP{}
	}
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			sleep(backoff << (i - 1))
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		err := sender.SendMail(addr, auth, cfg.From, cfg.To, msg)
		if err == nil {
			return nil
		}
		lastErr = err
		if !smtpShouldRetry(err) {
			break
		}
	}
	return lastErr
}

func smtpShouldRetry(err error) bool {
	if tpErr, ok := errors.AsType[*textproto.Error](err); ok {
		return tpErr.Code >= 400 && tpErr.Code < 500
	}
	return true // dial/network/TLS errors: assume transient
}

func buildEmailMessage(from string, to []string, p Payload) []byte {
	subject := fmt.Sprintf("[Argus] %s: %s", strings.ToUpper(p.Status), p.RuleName)
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	fmt.Fprintf(&buf, "Rule: %s\r\n", p.RuleName)
	fmt.Fprintf(&buf, "Status: %s\r\n", p.Status)
	fmt.Fprintf(&buf, "Severity: %s\r\n", p.Severity)
	fmt.Fprintf(&buf, "Query: %s\r\n", p.Query)
	fmt.Fprintf(&buf, "Comparator: %s\r\n", p.Comparator)
	fmt.Fprintf(&buf, "Threshold: %v\r\n", p.Threshold)
	fmt.Fprintf(&buf, "Value: %v\r\n", p.Value)
	fmt.Fprintf(&buf, "At: %s\r\n", p.At.Format(time.RFC3339))
	return buf.Bytes()
}
