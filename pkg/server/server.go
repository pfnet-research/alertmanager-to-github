package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pfnet-research/alertmanager-to-github/pkg/notifier"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Notifier notifier.Notifier
}

func New(notifier notifier.Notifier) (*Server) {
	return &Server{
		Notifier: notifier,
	}
}

func (s *Server) Router() *gin.Engine {
	router := gin.Default()
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.POST("/v1/webhook", s.v1Webhook)

	return router
}

func (s *Server) v1Webhook(c *gin.Context) {
	payload := &types.WebhookPayload{}

	if err := c.ShouldBindJSON(payload); err != nil {
		log.Error().Err(err).Msg("error binding JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.TODO()
	if err := s.Notifier.Notify(ctx, payload, c.Request.URL.Query()); err != nil {
		log.Error().Err(err).Msg("error notifying")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
