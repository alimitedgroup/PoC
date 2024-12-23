package common

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	slogotel "github.com/remychantenay/slog-otel"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"os"
	"regexp"
	"runtime/debug"
	"sync"
	"time"
)

var NumRequests metric.Int64Counter

func getBuildInfo() (string, string) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("ReadBuildInfo failed")
		return "unknown", "unknown"
	}

	name := bi.Path

	version := "unknown"

	gitHashRegex := regexp.MustCompile(`^[a-f0-9]{40}$`)

	dirty := false
	for _, kv := range bi.Settings {
		switch kv.Key {
		case "vcs.revision":
			version = kv.Value
			if gitHashRegex.MatchString(version) {
				version = version[:7]
			}
		case "vcs.modified":
			dirty = kv.Value == "true"
		}
	}
	if dirty {
		version = version + "-dirty"
	}

	return name, version
}

// SetupOTelSDK sets up an observability pipeline for exporting data to an OpenTelemetry ingestor
// that accepts the OTLP protocol.
func SetupOTelSDK(ctx context.Context, otlpURL string) func(context.Context) {
	name, version := getBuildInfo()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
			semconv.ServiceInstanceID(uuid.New().String()),
		),
	)

	if err != nil {
		panic(err)
	}
	var conn *grpc.ClientConn
	if len(otlpURL) != 0 {
		conn, err = grpc.NewClient(
			otlpURL,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			panic(err)
		}
	} else {
		slog.WarnContext(ctx, "No OTLP url provided, observability data will not be collected")
	}
	var traceProvider *sdktrace.TracerProvider
	var logProvider *sdklog.LoggerProvider
	var meterProvider *sdkmetric.MeterProvider
	if conn != nil {
		{
			exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
			if err != nil {
				panic(err)
			}
			bsp := sdktrace.NewBatchSpanProcessor(exp)
			traceProvider = sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithResource(res),
				sdktrace.WithSpanProcessor(bsp),
			)
			otel.SetTracerProvider(traceProvider)
		}
		{
			exp, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
			if err != nil {
				panic(err)
			}
			processor := sdklog.NewBatchProcessor(exp)
			logProvider = sdklog.NewLoggerProvider(
				sdklog.WithResource(res),
				sdklog.WithProcessor(processor),
			)
			global.SetLoggerProvider(logProvider)
		}
		{
			exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
			if err != nil {
				panic(err)
			}
			meterProvider = sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(res),
				sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second))),
			)
			otel.SetMeterProvider(meterProvider)

			err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
			if err != nil {
				panic(err)
			}
		}
	}

	otelHandler := otelslog.NewHandler(name, otelslog.WithVersion(version))

	defaultLogger := slog.New(slogmulti.Fanout(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		slogotel.OtelHandler{
			Next: otelHandler,
		},
	))
	slog.SetDefault(defaultLogger)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	slog.Info("OpenTelemetry setup successful", "name", name, "version", version)

	return func(ctx context.Context) {
		wg := sync.WaitGroup{}
		if traceProvider != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := traceProvider.Shutdown(ctx)
				if err != nil {
					slog.Default().ErrorContext(ctx, "trace provider shutdown error: "+err.Error())
				}
			}()
		}
		if logProvider != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := logProvider.Shutdown(ctx)
				if err != nil {
					slog.Default().ErrorContext(ctx, "log provider shutdown error: "+err.Error())
				}
			}()
		}
		if meterProvider != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := meterProvider.Shutdown(ctx)
				if err != nil {
					slog.Default().ErrorContext(ctx, "meter provider shutdown error: "+err.Error())
				}
			}()
		}
		wg.Wait()
	}
}
