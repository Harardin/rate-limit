package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Harardin/rate-limit/internal/config"
	"github.com/Harardin/rate-limit/pkg/hc"
	"github.com/Harardin/rate-limit/pkg/log"
	"github.com/Harardin/rate-limit/pkg/prometheus"
	"github.com/Harardin/rate-limit/pkg/rabbitbus"
)

type Server struct {
	// This is github.com/Harardin/rate-limit used just for example, most of services are unavailable

	msg chan string

	requestersList map[string]time.Time // in real better to use redis

	mx sync.RWMutex

	logger log.Logger
	config *config.Config
	hc     *hc.Server
	pm     *prometheus.Server

	// rabbit service
	rabbitService *rabbitbus.Service
}

func New(logger log.Logger, cfg *config.Config) (*Server, error) {
	s := &Server{
		logger:         logger,
		config:         cfg,
		msg:            make(chan string, 1),
		requestersList: make(map[string]time.Time),
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	defer s.Stop()

	return s.StartRateLimiterHTTP(ctx)
}

// this is limiter function example
func (s *Server) StartRateLimiterHTTP(ctx context.Context) error {

	http.HandleFunc("/req", s.HandleRequest)

	return http.ListenAndServe(":20001", nil)
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	s.mx.RLock()
	if t, ex := s.requestersList[ip]; ex && time.Since(t) < time.Second {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		s.mx.RUnlock()
		return
	}
	s.mx.RUnlock()

	switch r.Method {
	case "POST":
		// storing client IP
		s.mx.Lock()
		s.requestersList[ip] = time.Now()
		s.mx.Unlock()

		t := &http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Body:          io.NopCloser(bytes.NewBufferString("OK")),
			ContentLength: int64(len("OK")),
			Request:       r,
			Header:        make(http.Header, 0),
		}
		fmt.Fprint(w, t)
		return
	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) Stop() {
	// stop rabbit
	if s.rabbitService != nil {
		if err := s.rabbitService.CloseRabbitMQConnection(); err != nil {
			s.logger.Errorf("failed to stop rabbit: %v", err)
		}
	}

	// stop hc
	if s.hc != nil {
		s.hc.Stop(context.Background())
	}

	// stop prometheus
	if s.pm != nil {
		s.pm.Stop(context.Background())
	}

	s.logger.Info("server stopped")
}

func (s *Server) startHealthCheckServer() {
	// Init HC Server
	s.hc = hc.NewServer(s.logger, s.config.HealthCheck)

	// Register services
	s.hc.RegisterService(s.config.ServiceName, hc.NewService(0, nil, nil))

	// Start HC Server
	go s.hc.Start()
}
