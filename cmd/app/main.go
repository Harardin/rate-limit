package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Harardin/rate-limit/internal/config"
	"github.com/Harardin/rate-limit/internal/server"
	"github.com/Harardin/rate-limit/pkg/initialconfig"
	"github.com/Harardin/rate-limit/pkg/log"
)

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Init logger
	logger := log.New()
	defer logger.Sync()

	// Loading service config
	cfg := new(config.Config)
	configChangedEnvsCh := initialconfig.LoadConfig(logger, cfg)

	// Init Server
	srv, err := server.New(logger, cfg)
	if err != nil {
		logger.Fatalf("init server error: %v", err)
	}

	// Start server
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go startServer(logger, cfg, sig, wg, srv, configChangedEnvsCh)

	// Wait system signals
	<-sig
	close(sig)

	wg.Wait()
}

func startServer(logger log.Logger, cfg *config.Config, sig <-chan os.Signal, wg *sync.WaitGroup, srv *server.Server, configChangedEnvsCh chan []string) {
	ctx, cancel := context.WithCancel(context.Background())

	// Graceful shutdown
	go func() {
		<-sig
		cancel()
		wg.Done()
	}()

	for {
		// Start server
		go func(ctx context.Context) {
			defer wg.Done()
			wg.Add(1)

			if err := srv.Start(ctx); err != nil {
				logger.Fatalf("start server error: %v", err)
			}
		}(ctx)

		// You can restart certain services based on environment names
		changedEnvs := <-configChangedEnvsCh

		logger.Infof("changed enviroments: %v", changedEnvs)

		cancel()

		ctx, cancel = context.WithCancel(context.Background())
	}
}
