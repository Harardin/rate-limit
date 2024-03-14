package rabbitbus

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/Harardin/rate-limit/pkg/log"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	mq "github.com/rabbitmq/amqp091-go"
)

type Service struct {
	logger log.Logger
	config Config

	conn *mq.Connection

	mu sync.RWMutex
}

type Config struct {
	User     string `json:"RABBIT_USER" secret:"true"`
	Pass     string `json:"RABBIT_PASS" secret:"true"`
	IsSecure bool   `json:"RABBIT_IS_SECURE"`
	// For local development
	Addr string `json:"RABBIT_ADDR"`
}

func (c *Config) Validate() error {
	return validation.ValidateStruct(
		c,
		validation.Field(&c.User, validation.Required),
		validation.Field(&c.Pass, validation.Required),
	)
}

func (c *Config) getDSN(addr string) string {
	scheme := "amqp"
	if c.IsSecure {
		scheme = "amqps"
	}

	return fmt.Sprintf("%s://%s:%s@%s", scheme, c.User, c.Pass, addr)
}

func NewBus(logger log.Logger, c Config, addr string) (*Service, error) {
	logger.Info("opening rabbitmq connection")

	var conn *mq.Connection

	if c.IsSecure {
		c, err := mq.DialTLS(c.getDSN(addr), &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			logger.Error("failed to open rabbitmq tls connection", err)
			return nil, err
		}
		conn = c
	} else {
		c, err := mq.Dial(c.getDSN(addr))
		if err != nil {
			logger.Error("failed to open rabbitmq connection", err)
			return nil, err
		}
		conn = c
	}

	return &Service{
		logger: logger,
		config: c,
		conn:   conn,
	}, nil
}

func (s *Service) NewRabbitMQConnection(c Config, addr string) error {
	s.logger.Info("opening new rabbitmq connection instead of the old one")
	conn, err := mq.Dial(c.getDSN(addr))
	if err != nil {
		s.logger.Error("failed to open new rabbit ma connection via dsn", c.getDSN(addr), err)
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn = conn
	s.config = c

	return nil
}

func (s *Service) CloseRabbitMQConnection() error {
	s.logger.Info("closing rabbitmq connection")
	if s.conn == nil {
		return nil
	}

	return s.conn.Close()
}
