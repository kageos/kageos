package service

import (
	"context"
)

type ResolvedRecipient struct {
	Username string
	Email    string
}

type NotificationTarget struct {
	Recipient  ResolvedRecipient
	Channel    string
	WebhookURL string
	Secret     string
	Metadata   map[string]string
}

type ChannelProvider interface {
	Channel() string
	Deliver(ctx context.Context, target NotificationTarget, card NotificationCard) error
}

// InboxChannelProvider is kept for compatibility with early call sites. Inbox
// persistence is the message-server main path, not an external provider.
type InboxChannelProvider struct{}

func (InboxChannelProvider) Channel() string {
	return "inbox"
}

func (InboxChannelProvider) Deliver(context.Context, NotificationTarget, NotificationCard) error {
	return nil
}
