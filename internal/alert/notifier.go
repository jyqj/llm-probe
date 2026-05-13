package alert

import "log/slog"

// Notifier dispatches alert events to configured channels.
type Notifier struct {
	webhooks []WebhookSender
	logger   *slog.Logger
}

// NewNotifier creates a notifier with the given webhook destinations.
func NewNotifier(dests []WebhookDest, logger *slog.Logger) *Notifier {
	var senders []WebhookSender
	for _, d := range dests {
		senders = append(senders, NewWebhookSender(d))
	}
	return &Notifier{webhooks: senders, logger: logger}
}

// Notify sends an alert event to all configured channels.
func (n *Notifier) Notify(ev *Event) {
	for _, wh := range n.webhooks {
		if err := wh.Send(ev); err != nil {
			n.logger.Warn("webhook notification failed",
				"webhook", wh.Name(), "event", ev.ID, "error", err)
		} else {
			n.logger.Info("webhook notification sent",
				"webhook", wh.Name(), "event", ev.ID, "rule", ev.RuleName)
		}
	}
	ev.Notified = true
}

// NotifyAll sends multiple events.
func (n *Notifier) NotifyAll(events []*Event) {
	for _, ev := range events {
		if !ev.Silenced {
			n.Notify(ev)
		}
	}
}
