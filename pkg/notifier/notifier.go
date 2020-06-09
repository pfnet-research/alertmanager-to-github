package notifier

import (
	"context"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"net/url"
)

type Notifier interface {
	Notify(context.Context, *types.WebhookPayload, url.Values) error
}
