package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/puzpuzpuz/xsync/v3"
)

func StockUpdateHandler(_ context.Context, s *common.Service[ApiGatewayState], msg jetstream.Msg) error {
	slog.Info("Stock Update Handler", "subject", msg.Subject())

	var req messages.StockUpdate
	err := json.Unmarshal(msg.Data(), &req)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal stock update: %w", err)
		err2 := msg.TermWithReason(fmt.Sprintf("Failed to unmarshal stock update: %v", err))
		if err2 != nil {
			return fmt.Errorf(
				"while handling %w, another error happened: %w",
				err,
				fmt.Errorf("failed to term message: %w", err2),
			)
		}
		return err
	}

	warehouseId, found := strings.CutPrefix(msg.Subject(), "stock_updates.")
	if !found {
		err2 := msg.TermWithReason("received message on stock_updates with strange subject")
		if err2 != nil {
			return fmt.Errorf(
				"while handling 'received message on stock_updates with strange subject', another error happened: %w",
				fmt.Errorf("failed to Term message: %w", err2),
			)
		}
		return fmt.Errorf("received message on stock_updates with strange subject: %s", msg.Subject())
	}

	for _, row := range req {
		s.State().stock.Compute(warehouseId, func(oldValue *xsync.MapOf[string, int], loaded bool) (newValue *xsync.MapOf[string, int], delete bool) {
			if !loaded {
				oldValue = xsync.NewMapOf[string, int]()
			}
			// old, _ := oldValue.LoadOrCompute(row.GoodId, func() int { return 0 })
			newValue = oldValue
			newValue.Store(row.GoodId, row.Amount)
			return
		})
	}

	return nil
}

func StockGetRoute(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		stock, ok := s.State().stock.Load(c.Param("warehouseId"))
		if !ok {
			c.String(404, "Not Found")
			return
		}

		stock2 := map[string]int{}
		stock.Range(func(key string, value int) bool {
			stock2[key] = value
			return true
		})
		c.JSON(http.StatusOK, stock2)
	}
}

func StockPostRoute(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		warehouseId := c.Param("warehouseId")

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return
		}

		err = s.NatsConn().Publish(
			fmt.Sprintf("warehouse.add_stock.%s", warehouseId),
			body,
		)
		if err != nil {
			return
		}
	}
}

func WarehouseListRoute(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		keys := []string{}
		s.State().stock.Range(func(key string, _ *xsync.MapOf[string, int]) bool {
			keys = append(keys, key)
			return true
		})
		c.JSON(http.StatusOK, keys)
	}
}
