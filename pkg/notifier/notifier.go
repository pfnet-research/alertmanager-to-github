package notifier

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
)

type Notifier interface {
	Notify(context.Context, *types.WebhookPayload, gin.Params) error
}
