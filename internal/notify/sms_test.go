package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cragone/Super-Argus/internal/models"
	"github.com/stretchr/testify/require"
)

func smsChannel(t *testing.T, s *Sealer, to ...string) models.NotificationChannel {
	cfg := SMSConfig{AccountSID: "ACxxxx", From: "+15550000000", To: to}
	enc, err := s.Seal("authtoken")
	require.NoError(t, err)
	cfg.AuthTokenEnc = enc
	b, _ := json.Marshal(cfg)
	return models.NotificationChannel{Name: "c", Type: "sms", Config: string(b)}
}

func TestSMSSendsOneRequestPerRecipient(t *testing.T) {
	var gotUser, gotPass string
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		gotUser, gotPass, _ = r.BasicAuth()
		require.NoError(t, r.ParseForm())
		require.NotEmpty(t, r.Form.Get("Body"))
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	s := NewSealer("k")
	sms := &SMS{Sealer: s, Sleep: func(time.Duration) {}, BaseURL: srv.URL}
	ch := smsChannel(t, s, "+15551111111", "+15552222222")
	require.NoError(t, sms.Send(t.Context(), ch, Payload{Status: "firing", RuleName: "disk-full"}))
	require.Equal(t, 2, calls)
	require.Equal(t, "ACxxxx", gotUser)
	require.Equal(t, "authtoken", gotPass)
}

func TestSMSRetriesThenGivesUp(t *testing.T) {
	calls, status := 0, http.StatusServiceUnavailable
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(status)
	}))
	defer srv.Close()

	s := NewSealer("k")
	var sleeps []time.Duration
	sms := &SMS{Sealer: s, Sleep: func(d time.Duration) { sleeps = append(sleeps, d) }, BaseURL: srv.URL}
	ch := smsChannel(t, s, "+15551111111")
	require.Error(t, sms.Send(t.Context(), ch, Payload{}))
	require.Equal(t, 3, calls, "retried through all attempts")
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second}, sleeps)

	calls, status = 0, http.StatusBadRequest
	require.Error(t, sms.Send(t.Context(), ch, Payload{}))
	require.Equal(t, 1, calls, "4xx is not retried")
}
