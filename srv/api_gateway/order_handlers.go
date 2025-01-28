package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
)

func OrderCreateHandler(_ context.Context, s *common.Service[ApiGatewayState], msg jetstream.Msg) error {
	slog.Info("Order create Handler", "subject", msg.Subject())

	var req messages.OrderCreated
	err := json.Unmarshal(msg.Data(), &req)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal order created: %w", err)
		err2 := msg.TermWithReason(fmt.Sprintf("Failed to unmarshal order created: %v", err))
		if err2 != nil {
			return fmt.Errorf(
				"while handling %w, another error happened: %w",
				err,
				fmt.Errorf("failed to term message: %w", err2),
			)
		}
		return err
	}

	s.State().orders.Store(req.ID.String(), req)

	return nil
}

func OrderGetRoute(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		order, ok := s.State().orders.Load(c.Param("orderId"))
		if !ok {
			c.String(404, "Not Found")
			return
		}

		c.JSON(http.StatusOK, order)
	}
}

func OrderPostRoute(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		r, err := s.NatsConn().Request(
			"order.create",
			body,
			time.Second*2,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, map[string]any{"response": string(r.Data)})
	}
}

func OrderListRoute(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		keys := []string{}
		s.State().orders.Range(func(key string, _ messages.OrderCreated) bool {
			keys = append(keys, key)
			return true
		})
		c.JSON(http.StatusOK, keys)
	}
}
