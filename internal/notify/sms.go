package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cragone/Super-Argus/internal/models"
)

// SMSConfig is the JSON stored in NotificationChannel.Config for type "sms".
// AuthTokenEnc is the Sealer-encrypted Twilio auth token.
type SMSConfig struct {
	AccountSID   string   `json:"account_sid"`
	AuthTokenEnc string   `json:"auth_token_enc,omitempty"`
	From         string   `json:"from"`
	To           []string `json:"to"`
}

// SMS delivers payloads as text messages via Twilio's Messages REST API
// (https://www.twilio.com/docs/sms/api), one request per recipient. 5xx, 429
// and network errors are retried; other 4xx responses fail immediately, same
// policy as Webhook.
type SMS struct {
	Client   *http.Client
	Sealer   *Sealer
	Attempts int                 // default 3
	Backoff  time.Duration       // first delay, doubled each retry; default 1s
	Sleep    func(time.Duration) // injectable for tests
	BaseURL  string              // injectable for tests; defaults to the real Twilio API
}

const twilioBaseURL = "https://api.twilio.com"

// Send implements Notifier.
func (s *SMS) Send(ctx context.Context, ch models.NotificationChannel, p Payload) error {
	var cfg SMSConfig
	if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil || cfg.AccountSID == "" || cfg.From == "" || len(cfg.To) == 0 {
		return errors.New("channel has no valid sms config")
	}
	var token string
	if cfg.AuthTokenEnc != "" && s.Sealer != nil {
		t, err := s.Sealer.Open(cfg.AuthTokenEnc)
		if err != nil {
			return errors.New("cannot decrypt channel secret")
		}
		token = t
	}
	body := smsBody(p)

	base := s.BaseURL
	if base == "" {
		base = twilioBaseURL
	}
	target := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", base, cfg.AccountSID)

	attempts, backoff, sleep := s.Attempts, s.Backoff, s.Sleep
	if attempts <= 0 {
		attempts = 3
	}
	if backoff <= 0 {
		backoff = time.Second
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	for _, to := range cfg.To {
		form := url.Values{"From": {cfg.From}, "To": {to}, "Body": {body}}
		var lastErr error
		ok := false
		for i := 0; i < attempts; i++ {
			if i > 0 {
				sleep(backoff << (i - 1))
				if ctx.Err() != nil {
					return ctx.Err()
				}
			}
			retry, err := s.post(ctx, client, target, cfg.AccountSID, token, form)
			if err == nil {
				ok = true
				break
			}
			lastErr = err
			if !retry {
				break
			}
		}
		if !ok {
			return lastErr
		}
	}
	return nil
}

func (s *SMS) post(ctx context.Context, c *http.Client, target, sid, token string, form url.Values) (retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(sid, token)
	resp, err := c.Do(req)
	if err != nil {
		return true, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode/100 == 2:
		return false, nil
	case resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests:
		return true, fmt.Errorf("twilio returned %d", resp.StatusCode)
	default:
		return false, fmt.Errorf("twilio returned %d", resp.StatusCode)
	}
}

func smsBody(p Payload) string {
	return fmt.Sprintf("[Argus] %s %s: %s %s %v (value %v)",
		strings.ToUpper(p.Status), p.Severity, p.RuleName, p.Comparator, p.Threshold, p.Value)
}
