package hc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Harardin/rate-limit/pkg/log"

	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
)

type Config struct {
	// Port - default 10002
	Port string `default:"10002" json:"HEALTH_CHECK_PORT"`
	// Endpoint - default /hc
	Endpoint string `default:"/hc" json:"HEALTH_CHECK_ENDPOINT"`
	// InActive - default false
	InActive bool `json:"HEALTH_CHECK_IN_ACTIVE"`
}

type Server struct {
	logger log.Logger
	config Config

	mu       sync.Mutex
	services map[string]*Service

	srv *http.Server
}

type CheckFunc func() error

type Service struct {
	// Current service code. 0 - all is fine; 1 - some problems
	Code int
	// Time between new service checks. Default 10 seconds
	CheckTimeout time.Duration
	// Service check function
	CheckFunc CheckFunc
	ctx       context.Context
	cancel    context.CancelFunc

	// Hook after successful execution of CheckFunc
	CheckFuncAfterHook CheckFunc
}

type Services map[string]*Service

func NewServer(logger log.Logger, config Config) *Server {
	return &Server{
		logger:   logger,
		config:   config,
		services: make(Services),
	}
}

// NewService return new service
//
// Default timeout: 10 seconds
func NewService(timeout int, checkFunc CheckFunc, checkFuncAfterHook CheckFunc) *Service {
	if timeout == 0 {
		timeout = 10
	}

	return &Service{
		CheckTimeout:       time.Duration(timeout) * time.Second,
		CheckFunc:          checkFunc,
		CheckFuncAfterHook: checkFuncAfterHook,
	}
}

// Start health check server
//
// Documentation available there
// https://yt.heronodes.io/articles/FP-A-15/Healthcheck-requires
func (s *Server) Start() {
	if s.config.InActive {
		return
	}

	port := s.config.Port
	if port == "" {
		port = "10002"
	}

	endpoint := s.config.Endpoint
	if endpoint == "" {
		endpoint = "/hc"
	}

	r := mux.NewRouter()
	r.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		res, err := json.Marshal(s.GetServicesCode())
		if err != nil {
			s.logger.Errorf("marshal hc response error: %v", err)
			return
		}

		if _, err := w.Write(res); err != nil {
			s.logger.Errorf("handle health check error: %v", err)
		}
	})

	s.srv = &http.Server{Addr: ":" + port, Handler: r}

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Fatalf("failed to start health check server on port %s: %v", port, err)
	}
}

func (s *Server) RegisterService(serviceName string, service *Service) {
	existService, _ := s.GetService(serviceName)
	if existService != nil {
		s.DeleteService(serviceName)
	}

	// register checking func
	if service.CheckFunc != nil {
		ctx, cancel := context.WithCancel(context.Background())
		service.ctx = ctx
		service.cancel = cancel

		if service.CheckTimeout == 0 {
			service.CheckTimeout = time.Second * 10
		}

		go func(serivce *Service) {
			for {
				select {
				case <-time.After(serivce.CheckTimeout):
					// Execute checking func
					if err := service.CheckFunc(); err != nil {
						s.UpdateServiceCode(serviceName, 1)
						s.logger.Errorf("check service %s failed with error: %v", serviceName, err)
					} else {
						s.UpdateServiceCode(serviceName, 0)
					}

					// Execute after hook
					if service.CheckFuncAfterHook != nil {
						if err := service.CheckFuncAfterHook(); err != nil {
							s.logger.Errorf("check service after hook %s failed with error: %v", serviceName, err)
						}
					}
					continue
				case <-service.ctx.Done():
					return
				}
			}
		}(service)
	}

	s.mu.Lock()
	s.services[serviceName] = service
	s.mu.Unlock()
}

func (s *Server) UpdateServiceCode(serviceName string, code int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	service, ok := s.services[serviceName]
	if !ok {
		return
	}
	service.Code = code
}

func (s *Server) DeleteService(serviceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	service, ok := s.services[serviceName]
	if !ok {
		return
	}

	if service.cancel != nil {
		service.cancel()
	}

	delete(s.services, serviceName)
}

func (s *Server) GetServicesCode() map[string]int {
	res := make(map[string]int)

	s.mu.Lock()
	for name, service := range s.services {
		res[name] = service.Code
	}
	s.mu.Unlock()

	return res
}

func (s *Server) GetServiceCode(serviceName string) (int, error) {
	service, ok := s.services[serviceName]
	if !ok {
		return 0, fmt.Errorf("service %s does not exist", serviceName)
	}

	return service.Code, nil
}

func (s *Server) GetService(serviceName string) (*Service, error) {
	service, ok := s.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s does not exist", serviceName)
	}

	return service, nil
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Errorf("failed to stop health check http server: %v", err)
	}
}
