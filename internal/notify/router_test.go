package notify

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/cragone/Super-Argus/internal/models"
	"github.com/stretchr/testify/require"
)

func TestRouterRejectsUnknownType(t *testing.T) {
	r := &Router{}
	err := r.Send(t.Context(), models.NotificationChannel{Type: "bogus"}, Payload{})
	require.ErrorContains(t, err, "unknown channel type")
}

func TestRouterReportsMissingSender(t *testing.T) {
	r := &Router{}
	for _, typ := range []string{"webhook", "email", "sms"} {
		err := r.Send(t.Context(), models.NotificationChannel{Type: typ}, Payload{})
		require.ErrorContains(t, err, "no "+typ+" sender configured", typ)
	}
}

func TestRouterDispatchesToWebhook(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewSealer("k")
	r := &Router{Webhook: &Webhook{Sealer: s, Sleep: func(time.Duration) {}}}
	require.NoError(t, r.Send(t.Context(), channel(t, s, srv.URL, ""), Payload{}))
	require.True(t, called, "Router.Send must reach the configured Webhook sender")
}

func TestRouterDispatchesToEmail(t *testing.T) {
	srv := startFakeSMTP(t)
	host, portStr, _ := net.SplitHostPort(srv.addr())
	port, _ := strconv.Atoi(portStr)

	s := NewSealer("k")
	r := &Router{Email: &Email{Sealer: s, Sleep: func(time.Duration) {}}}
	require.NoError(t, r.Send(t.Context(), emailChannel(t, s, host, port, ""), Payload{RuleName: "disk-full"}))
	require.NotEmpty(t, srv.bodies, "Router.Send must reach the configured Email sender")
}

func TestRouterDispatchesToSMS(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	s := NewSealer("k")
	r := &Router{SMS: &SMS{Sealer: s, Sleep: func(time.Duration) {}, BaseURL: srv.URL}}
	require.NoError(t, r.Send(t.Context(), smsChannel(t, s, "+15551111111"), Payload{}))
	require.True(t, called, "Router.Send must reach the configured SMS sender")
}
