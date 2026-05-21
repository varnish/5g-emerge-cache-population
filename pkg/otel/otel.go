package otel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	logglobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
)

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, scrapeInterval time.Duration) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	res, err := resource.New(
		context.Background(),
		resource.WithFromEnv(),      // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(), // Discover and provide information about the OpenTelemetry SDK used.
		resource.WithProcess(),      // Discover and provide process information.
		resource.WithOS(),           // Discover and provide OS information.
		resource.WithContainer(),    // Discover and provide container information.
		resource.WithHost(),         // Discover and provide host information.
		// resource.WithDetectors(thirdparty.Detector{}), // Bring your own external Detector implementation.
	)
	if err != nil {
		handleErr(err)
		return
	}

	// logger provider.
	loggerProvider, err := newLoggerProvider(res)
	if err != nil {
		handleErr(err)
		return
	}
	if loggerProvider != nil {
		logglobal.SetLoggerProvider(loggerProvider)
		shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	}

	// meter provider
	meterProvider, err := newMeterProvider(res, scrapeInterval)
	if err != nil {
		handleErr(err)
		return
	}
	if meterProvider != nil {
		otel.SetMeterProvider(meterProvider)
		shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	}

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

type proto int

const (
	protoError proto = iota
	protoNone
	protoConsole
	protoHttp
	protoGrpc
)

func newMeterProvider(res *resource.Resource, scrapeInterval time.Duration) (meterProvider *metric.MeterProvider, err error) {
	var exp metric.Exporter

	protocol, err := pickProtocol("METRICS")
	if err != nil {
		return nil, err
	}
	switch protocol {
	case protoNone:
		return nil, nil
	case protoConsole:
		exp, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	case protoHttp:
		exp, err = otlpmetrichttp.New(context.Background())
	case protoGrpc:
		exp, err = otlpmetricgrpc.New(context.Background())
	default:
		return nil, fmt.Errorf("unexpected protocol value: %d", protocol)
	}
	if err != nil {
		return nil, err
	}
	meterProvider = metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exp, metric.WithInterval(scrapeInterval))),
		metric.WithResource(res),
	)
	return meterProvider, nil
}

func newLoggerProvider(res *resource.Resource) (loggerProvider *sdklog.LoggerProvider, err error) {
	var exp sdklog.Exporter

	protocol, err := pickProtocol("LOGS")
	if err != nil {
		return nil, err
	}
	switch protocol {
	case protoNone:
		return nil, nil
	case protoConsole:
		exp, err = stdoutlog.New(stdoutlog.WithPrettyPrint())
	case protoHttp:
		exp, err = otlploghttp.New(context.Background())
	case protoGrpc:
		exp, err = otlploggrpc.New(context.Background())
	default:
		return nil, fmt.Errorf("unexpected protocol value: %d", protocol)
	}
	if err != nil {
		return nil, err
	}

	processor := sdklog.NewBatchProcessor(exp)
	loggerProvider = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)
	return loggerProvider, nil
}

func pickProtocol(module string) (proto, error) {
	var env_name_exp, env_name_pro, exp, pro string

	generic_exp := "OTEL_EXPORTER"
	specific_exp := "OTEL_" + module + "_EXPORTER"
	generic_pro := "OTEL_EXPORTER_OTLP_PROTOCOL"
	specific_pro := "OTEL_EXPORTER_OTLP_" + module + "_PROTOCOL"

	if os.Getenv(specific_exp) != "" {
		exp = os.Getenv(specific_exp)
		env_name_exp = specific_exp
	} else {
		exp = os.Getenv(generic_exp)
		env_name_exp = generic_exp
	}
	if os.Getenv(specific_pro) != "" {
		pro = os.Getenv(specific_pro)
		env_name_pro = specific_pro
	} else {
		pro = os.Getenv(generic_pro)
		env_name_pro = generic_pro
	}

	switch exp {
	case "console":
		return protoConsole, nil
	case "none":
		return protoNone, nil
	case "otlp", "":
		switch pro {
		case "http/protobuf":
			return protoHttp, nil
		case "grpc", "":
			return protoGrpc, nil
		default:
			return protoError, fmt.Errorf(`%s environment variable has invalid value "%s" (can only be empty, "http/protobuf" or "grpc")`, env_name_pro, pro)
		}
	default:
		return protoError, fmt.Errorf(`%s environment variable has invalid value "%s" (can only be empty, "none", "console", or "otlp" (default))`, env_name_exp, exp)
	}
}
