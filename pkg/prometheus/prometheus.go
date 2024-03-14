package prometheus

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/Harardin/rate-limit/pkg/log"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "github.com/mitchellh/mapstructure"
)

type Config struct {
	// Port - default 10001
	Port string `json:"PROMETHEUS_PORT" default:"10001"`
	// Endpoint - default /metrics
	Endpoint string `json:"PROMETHEUS_ENDPOINT" default:"/metrics"`
	// Disabled - default false
	Disabled bool `json:"PROMETHEUS_DISABLED"`
}

type Server struct {
	logger log.Logger
	config Config

	srv *http.Server

	requestCounter *prometheus.CounterVec
}

func NewServer(logger log.Logger, config Config, serviceName string) *Server {
	if serviceName == "" {
		logger.Errorf("prometheus error: service name is empty")
	}

	return &Server{
		logger: logger,
		config: config,
		// TODO: make it configurable, like hc
		requestCounter: promauto.With(prometheus.NewRegistry()).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: strings.ReplaceAll(serviceName, "-", "_"),
				Name:      "requests_counter",
				Help:      "",
			}, []string{"query", "status"}),
	}
}

// Start prometheus server
func (s *Server) Start(ctx context.Context) {
	go func() {
		if s.config.Disabled {
			return
		}

		port := s.config.Port
		if port == "" {
			port = "10001"
		}

		endpoint := s.config.Endpoint
		if endpoint == "" {
			endpoint = "/metrics"
		}

		r := mux.NewRouter()
		r.Path(endpoint).Handler(promhttp.Handler())

		s.srv = &http.Server{Addr: ":" + port, Handler: r}

		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("failed to start prometheus on port %s: %v", port, err)
		}
	}()
}

func (s *Server) IncrementRequestsCount(query, result string) {
	s.requestCounter.WithLabelValues(query, result).Inc()
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Errorf("failed to stop prometheus http server: %v", err)
	}
}
