package main

import (
	"context"
	"github.com/alimitedgroup/palestra_poc/common"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	var err error
	common.NumRequests, err = otel.Meter("").Int64Counter("num_requests")
	require.NoError(t, err)

	s := natsserver.RunRandClientPortServer()
	nc, err := nats.Connect(s.ClientURL())
	require.NoError(t, err)

	// mock, err := pgxmock.NewPool()
	// require.NoError(t, err)
	// defer mock.Close()

	catalog, err := setupCatalog(context.Background(), nc, nil)
	require.NoError(t, err)
	defer catalog.Stop()

	// mock.ExpectQuery("SELECT id, name FROM catalog WHERE id = $1").
	// 	WithArgs(1).
	// 	WillReturnRows(
	// 		pgxmock.NewRows([]string{"id", "name"}).AddRow(1, "catalog_item_name"),
	// 	)

	resp, err := nc.Request("catalog.ping", nil, 1*time.Second)
	require.NoError(t, err)
	require.Equal(t, resp.Data, []byte("pong"))
}
