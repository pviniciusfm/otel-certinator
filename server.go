package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

type server struct {
	http.Server
	logger                *zap.Logger
	router                *http.ServeMux
	tracer                trace.Tracer
	appPort               int
	serverShutdownTimeout time.Duration
	flushTraceCB          func()
	serviceName           string
}

type Server interface {
	Start() error
	AddHandlerFunc(route string, handlFunc http.HandlerFunc)
}

var (
	requestsCounterKey = label.Key("request_count")
	errorCount         = label.Key("error_count")
	statusCodeKey      = label.Key("status_code")
	requestTypeKey     = label.Key("request_type")
	responseLatency    = label.Key("request_latency")
	envKey             = label.Key("development")
	commonLabels       = []label.KeyValue{envKey.String("ecomdev")}
)

func NewServer(serviceName string, logger *zap.Logger, tracer trace.Tracer, port int) Server {
	sv := &server{
		serviceName:           serviceName,
		router:                http.NewServeMux(),
		appPort:               port,
		serverShutdownTimeout: time.Second * 30,
		logger:                logger,
		tracer:                tracer,
	}
	sv.flushTraceCB = sv.initTracingProvider()
	sv.initMeter()
	return sv
}

// GetTracer returns global tracer
func (s *server) GetTracer() trace.Tracer {
	return s.tracer
}

// start http server
func (s *server) Start() error {
	go s.initSignals()
	s.logger.Info("Initializing http server", zap.Int("http.port", s.appPort))
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", s.appPort), s.router)
}

// init signals
func (s *server) initSignals() {
	var captureSignal = make(chan os.Signal, 1)
	signal.Notify(captureSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	s.signalHandler(<-captureSignal)
}

//AddHandlerFunc wraps handler func into otel handler
func (s *server) AddHandlerFunc(route string, handleFunc http.HandlerFunc) {
	s.router.Handle(route, otelhttp.NewHandler(http.HandlerFunc(handleFunc), route))
}

// HandleError checks if the error is not nil, writes it to the output
// with the specified status code, and returns true. If error is nil it returns false.
func HandleError(w http.ResponseWriter, err error, statusCode int) bool {
	if err == nil {
		return false
	}
	http.Error(w, err.Error(), statusCode)
	return true
}

// signal handler
func (s *server) signalHandler(signal os.Signal) {
	s.logger.Info("caught signal", zap.String("signal", signal.String()))
	s.logger.Info("wait for 1 second to finish processing")
	time.Sleep(1 * time.Second)

	switch signal {
	case syscall.SIGHUP:
	case syscall.SIGINT:
	case syscall.SIGTERM:
	case syscall.SIGQUIT:
	default:
		s.logger.Info("os term signal captured shutting down http server...")
	}
	s.flushTraceCB()
	s.logger.Info("finished server cleanup")
	os.Exit(0)
}

// initTracingProvider initializes opentelemetry collector
func (s *server) initTracingProvider() func() {
	// Create and install Jaeger export pipeline
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://localhost:16686"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: s.serviceName,
			Tags:        commonLabels,
		}),
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	if err != nil {
		s.logger.Fatal("could not initialize jaeger", zap.Error(err))
	}
	return func() {
		flush()
	}
}

// initMeter initializes prometheus exporter
func (s *server) initMeter() {
	s.logger.Info("initializing prometheus in route /metrics")
	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{})
	if err != nil {
		s.logger.Fatal("failed to initialize prometheus exporter %v", zap.Error(err))
	}
	s.router.HandleFunc("/metrics", exporter.ServeHTTP)
	global.SetMeterProvider(exporter.Provider())
}
