package common

import (
	"context"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/micro"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// NewInProcessNATSServer creates a temporary, in-process NATS server, and returns a connection to it. Useful for tests
func NewInProcessNATSServer(t *testing.T) *nats.Conn {
	tmp, err := os.MkdirTemp("", "nats_test")
	require.NoError(t, err)

	server, err := natsserver.NewServer(&natsserver.Options{
		DontListen: true,
		JetStream:  true,
		StoreDir:   tmp,
	})
	require.NoError(t, err)

	server.Start()
	t.Cleanup(func() {
		server.Shutdown()
		err := os.RemoveAll(tmp)
		require.NoError(t, err)
	})
	require.True(t, server.ReadyForConnections(1*time.Second))

	// Create a connection.
	conn, err := nats.Connect("", nats.InProcessServer(server))
	require.NoError(t, err)

	return conn
}

// StatefulHandler is a utility type for creating NATS handlers that get passed
// a context.Context, and some shared state that is necessary for handling the request.
//
// It also takes care of setting up observability for each request.
type StatefulHandler struct {
	ctx     context.Context
	handler func(context.Context, micro.Request, *pgxpool.Pool)
	db      *pgxpool.Pool
}

func (h StatefulHandler) Handle(req micro.Request) {
	NumRequests.Add(h.ctx, 1, metric.WithAttributes(
		attribute.String("subject", (req).Subject()),
	))

	// misurare i tempi di risposta, e salvarli in un histogram
	start := time.Now()

	h.handler(h.ctx, req, h.db)

	ResponseTime.Record(h.ctx, time.Since(start).Milliseconds(), metric.WithAttributes(
		attribute.String("subject", (req).Subject()),
	))
}

func NewHandler(ctx context.Context, db *pgxpool.Pool, handler func(context.Context, micro.Request, *pgxpool.Pool)) StatefulHandler {
	return StatefulHandler{ctx: ctx, handler: handler, db: db}
}
