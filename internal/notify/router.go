package notify

import (
	"context"
	"fmt"

	"github.com/cragone/Super-Argus/internal/models"
)

// Router dispatches Send to the sender matching ch.Type. It implements
// Notifier so it can be wired in wherever a single Notifier was used before.
type Router struct {
	Webhook *Webhook
	Email   *Email
	SMS     *SMS
}

// Send implements Notifier.
func (r *Router) Send(ctx context.Context, ch models.NotificationChannel, p Payload) error {
	switch ch.Type {
	case "webhook":
		if r.Webhook == nil {
			return fmt.Errorf("notify: no webhook sender configured")
		}
		return r.Webhook.Send(ctx, ch, p)
	case "email":
		if r.Email == nil {
			return fmt.Errorf("notify: no email sender configured")
		}
		return r.Email.Send(ctx, ch, p)
	case "sms":
		if r.SMS == nil {
			return fmt.Errorf("notify: no sms sender configured")
		}
		return r.SMS.Send(ctx, ch, p)
	default:
		return fmt.Errorf("notify: unknown channel type %q", ch.Type)
	}
}
