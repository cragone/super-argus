package notify

import (
	"net/http"
	"net/http/httptest"
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
