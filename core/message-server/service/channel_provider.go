package service

import (
	"context"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
)

type ResolvedRecipient struct {
	Username string
	Email    string
}

type ChannelProvider interface {
	Channel() string
	Deliver(ctx context.Context, entry *model.MessageEntry, payload dto.MessageSendPayload, recipient ResolvedRecipient) error
}

// InboxChannelProvider marks the default MVP channel. Inbox persistence happens
// before provider delivery, so this provider intentionally has no side effect.
type InboxChannelProvider struct{}

func (InboxChannelProvider) Channel() string {
	return "inbox"
}

func (InboxChannelProvider) Deliver(context.Context, *model.MessageEntry, dto.MessageSendPayload, ResolvedRecipient) error {
	return nil
}
