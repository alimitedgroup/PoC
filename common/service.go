package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Service collects and unifies various functionality that would otherwise be repeated among all services
//
// In particular, it implements cancellation with a context, automatic tracing of requests and responses
type Service[S any] struct {
	state           S
	ctx             context.Context
	nc              *nats.Conn
	js              jetstream.JetStream
	subscriptions   []*nats.Subscription
	subscriptionsJs []jetstream.ConsumeContext
}

// Handler represents a handler for a particular NATS Core subject
type Handler[S any] func(context.Context, *Service[S], *nats.Msg)

// JsHandler represents a handler for a particular JetStream stream
type JsHandler[S any] func(context.Context, *Service[S], jetstream.Msg) error

// NewService will create a new instance of Service
func NewService[S any](ctx context.Context, nc *nats.Conn, state S) *Service[S] {
	js, err := jetstream.New(nc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create JetStream client", "error", err)
		return nil
	}

	s := &Service[S]{ctx: ctx, state: state, nc: nc, js: js}

	go func(ctx context.Context) {
		<-ctx.Done()

		for _, sub := range s.subscriptions {
			err := sub.Drain()
			if err != nil {
				slog.ErrorContext(ctx, "Failed to drain subscription", "error", err, "subject", sub.Subject)
			}
		}
	}(ctx)

	return s
}

// State returns the state of this service.
//
// Note that the state can be shared between multiple goroutines,
// so you should implement locking if it is needed.
func (s *Service[S]) State() *S {
	return &s.state
}

// NatsConn returns the NATS connection associated with this Service
func (s *Service[S]) NatsConn() *nats.Conn {
	return s.nc
}

// JetStream returns the JetStream connection associated with this Service
func (s *Service[S]) JetStream() jetstream.JetStream {
	return s.js
}

// RegisterHandler registers a handler for the given NATS subject
func (s *Service[S]) RegisterHandler(subject string, handler Handler[S]) {
	subscription, err := s.NatsConn().Subscribe(subject, func(msg *nats.Msg) {
		handler(s.ctx, s, msg)
	})
	if err != nil {
		slog.ErrorContext(s.ctx, "Failed to subscribe to subject", "subject", subject, "error", err)
		panic(err)
	}

	s.subscriptions = append(s.subscriptions, subscription)
}

// RegisterJsHandler registers a handler for the given JetStream stream
//
// Note that, if your handler returns an error, it is your responsibility to either Nak or Term the message.
// If, instead, no error is returned, then Ack gets automatically called.
func (s *Service[S]) RegisterJsHandler(subject string, handler JsHandler[S], opts ...JsHandlerOpt) {
	cfg := jetstream.ConsumerConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	consumer, err := s.JetStream().CreateConsumer(s.ctx, subject, cfg)
	if err != nil {
		slog.ErrorContext(s.ctx, "Failed to create consumer", "subject", subject, "error", err, "consumerConfig", cfg)
		panic(err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		err := handler(s.ctx, s, msg)
		if err != nil {
			slog.ErrorContext(s.ctx, "Failed to handle message", "subject", subject, "error", err, "msg", msg)
		} else {
			err = msg.Ack()
			if err != nil {
				slog.ErrorContext(s.ctx, "Failed to ack message", "subject", subject, "error", err, "msg", msg)
			}
		}
	})
	if err != nil {
		panic(err)
	}

	s.subscriptionsJs = append(s.subscriptionsJs, cc)
}

// RegisterJsHandlerExisting registers a handler for the given JetStream stream.
//
// An ad-hoc, ephemeral consumer will be created for the given stream using the given opts.
// After having read all messages (i.e. when a messages is returned with NumPending = 0),
// then the subscription will be automatically closed, and only then this function will return.
func (s *Service[S]) RegisterJsHandlerExisting(stream string, handler JsHandler[S], opts ...JsHandlerOpt) error {
	cfg := jetstream.ConsumerConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	consumer, err := s.js.CreateConsumer(s.ctx, stream, cfg)
	if err != nil {
		return fmt.Errorf("failed to create JetStream consumer: %w", err)
	}

	// Consume all messages, and stop when they are finished or an error occurs
	var cc jetstream.ConsumeContext
	var msgErr error

	// fetch consumer info
	info, err := consumer.Info(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to get consumer info: %w", err)
	}
	// if num pending is zero the stream have just been created and there are no messages to consume
	if info.NumPending == 0 {
		return nil
	}

	cc, err = consumer.Consume(func(msg jetstream.Msg) {
		msgErr = handler(s.ctx, s, msg)
		if msgErr != nil {
			err = fmt.Errorf("failed to handle message: %w", msgErr)
			cc.Stop()
		}

		var meta *jetstream.MsgMetadata
		meta, msgErr = msg.Metadata()
		if msgErr != nil {
			err = fmt.Errorf("failed to read message metadata: %w", msgErr)
			cc.Stop()
		}
		if msgErr == nil && meta.NumPending == 0 {
			cc.Drain()
		}
	})
	if err != nil {
		return fmt.Errorf("failed to consume from stream: %w", err)
	}

	<-cc.Closed()
	return msgErr
}

// JsHandlerOpt represents various options used when creating a JetStream handler
type JsHandlerOpt func(config *jetstream.ConsumerConfig)

// WithDeliverNew will set the consumer's DeliveryPolicy to DeliverNew
func WithDeliverNew() JsHandlerOpt {
	return func(config *jetstream.ConsumerConfig) {
		config.DeliverPolicy = jetstream.DeliverNewPolicy
	}
}

// WithDeliverAll will set the consumer's DeliveryPolicy to DeliverAll
func WithDeliverAll() JsHandlerOpt {
	return func(config *jetstream.ConsumerConfig) {
		config.DeliverPolicy = jetstream.DeliverAllPolicy
	}
}

// WithSubjectFilter will filter the delivered messages to those specified. Mutually exclusive with WithSubjectsFilter
func WithSubjectFilter(subject string) JsHandlerOpt {
	return func(config *jetstream.ConsumerConfig) {
		config.FilterSubject = subject
	}
}

// WithSubjectsFilter will filter the delivered messages to those specified. Mutually exclusive with WithSubjectFilter
func WithSubjectsFilter(subjects []string) JsHandlerOpt {
	return func(config *jetstream.ConsumerConfig) {
		config.FilterSubjects = append(config.FilterSubjects, subjects...)
	}
}
