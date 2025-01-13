package common

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

func TestService_RegisterJsHandlerExisting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	t.Cleanup(cancel)

	nc := NewInProcessNATSServer(t)
	t.Cleanup(func() { nc.Close() })

	js, err := jetstream.New(nc)
	require.NoError(t, err)

	_, err = js.CreateStream(ctx, jetstream.StreamConfig{Name: "test"})
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		_, err = js.Publish(ctx, "test", []byte(fmt.Sprintf("%d", i)))
		require.NoError(t, err)
	}

	service := NewService(ctx, nc, struct{}{})

	prev := -1
	err = service.RegisterJsHandlerExisting("test", func(ctx context.Context, s *Service[struct{}], msg jetstream.Msg) error {
		id, err := strconv.Atoi(string(msg.Data()))
		require.NoError(t, err)
		require.Equal(t, prev+1, id)
		prev = id

		err = msg.Ack()
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 99, prev)
}

func TestService_RegisterHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	t.Cleanup(cancel)

	nc := NewInProcessNATSServer(t)
	t.Cleanup(nc.Close)

	type state struct{ s []string }
	service := NewService(ctx, nc, state{})
	service.RegisterHandler("subject", func(ctx context.Context, s *Service[state], msg *nats.Msg) {
		s.State().s = append(s.State().s, string(msg.Data))
	})

	require.NoError(t, nc.Publish("subject", []byte("hello")))
	require.NoError(t, nc.Publish("subject", []byte("world")))

	time.Sleep(500 * time.Millisecond)

	require.Equal(t, []string{"hello", "world"}, service.State().s)
}

func TestService_RegisterJsHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	t.Cleanup(cancel)

	nc := NewInProcessNATSServer(t)
	t.Cleanup(nc.Close)

	type state struct{ s []string }
	service := NewService(ctx, nc, state{})
	_, err := service.JetStream().CreateStream(ctx, jetstream.StreamConfig{Name: "stream"})
	require.NoError(t, err)

	service.RegisterJsHandler("stream", func(ctx context.Context, s *Service[state], msg jetstream.Msg) error {
		s.State().s = append(s.State().s, string(msg.Data()))
		return nil
	})

	require.NoError(t, nc.Publish("stream", []byte("hello")))
	require.NoError(t, nc.Publish("stream", []byte("world")))

	time.Sleep(500 * time.Millisecond)

	require.Equal(t, []string{"hello", "world"}, service.State().s)
}

func TestService_Cleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	nc := NewInProcessNATSServer(t)
	t.Cleanup(nc.Close)

	require.Zero(t, nc.NumSubscriptions())

	svc := NewService(ctx, nc, struct{}{})
	svc.RegisterHandler("cleanup", func(ctx context.Context, s *Service[struct{}], msg *nats.Msg) {
		require.NoError(t, msg.Respond(msg.Data))
	})

	resp, err := nc.Request("cleanup", []byte("hello"), 50*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), resp.Data)

	cancel()

	time.Sleep(50 * time.Millisecond)

	resp, err = nc.Request("cleanup", []byte("hello"), 50*time.Millisecond)
	require.ErrorIs(t, err, nats.ErrNoResponders)
}
