package config

import (
	"fmt"

	"github.com/Harardin/rate-limit/pkg/consul"
	"github.com/Harardin/rate-limit/pkg/hc"
	"github.com/Harardin/rate-limit/pkg/postgres"
	"github.com/Harardin/rate-limit/pkg/prometheus"
	"github.com/Harardin/rate-limit/pkg/rabbitbus"
	"github.com/Harardin/rate-limit/pkg/redisclient"
	"github.com/Harardin/rate-limit/pkg/utils"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type DiscoveryConfig struct {
	PostgresAddrs consul.GetServiceAddressResponse `json:"POSTGRES_ADDRS"`
	RabbitAddrs   consul.GetServiceAddressResponse `json:"RABBIT_ADDRS"`
	RedisAddrs    consul.GetServiceAddressResponse `json:"REDIS_ADDRS"`
}

type GlobalConfig struct{}

type LocalConfig struct {
	ServiceName         string `json:"SERVICE_NAME" default:"github.com/Harardin/rate-limit"`
	StandName           string `json:"CONSUL_STAND_NAME" env:"CONSUL_STAND_NAME"`
	Rabbit              rabbitbus.Config
	Prometheus          prometheus.Config
	Postgres            postgres.Config
	Redis               redisclient.Config
	HealthCheck         hc.Config
	GpgPublicSignatures map[string]string `json:"GPG_PUBLIC_SIGNATURES"`

	// Discovery services
	// This items will be pass to the DiscoveryConfig
	DiscoveryPostgresService string `json:"DISCOVERY_POSTGRES_SERVICE" discovery:"POSTGRES_ADDRS"`
	DiscoveryRabbitService   string `json:"DISCOVERY_RABBIT_SERVICE" discovery:"RABBIT_ADDRS"`
	DiscoveryRedisService    string `json:"DISCOVERY_REDIS_SERVICE" discovery:"REDIS_ADDRS"`
}

type Config struct {
	DiscoveryConfig
	GlobalConfig
	LocalConfig
}

// Validate discovery config
func (c *DiscoveryConfig) Validate() error {
	return validation.ValidateStruct(
		c,
		validation.Field(&c.PostgresAddrs, validation.Required),
		validation.Field(&c.RabbitAddrs, validation.Required),
		validation.Field(&c.RedisAddrs, validation.Required),
	)
}

// Validate global config
func (c *GlobalConfig) Validate() error {
	return nil
}

// Validate local config
func (c *LocalConfig) Validate() error {
	// Validate postgres
	if err := c.Postgres.Validate(); err != nil {
		return err
	}

	// Validate rabbit
	if err := c.Rabbit.Validate(); err != nil {
		return err
	}

	// Validate prometheus
	if !c.Prometheus.Disabled {
		if err := validation.ValidateStruct(
			&c.Prometheus,
			validation.Field(&c.Prometheus.Port, validation.Required),
			validation.Field(&c.Prometheus.Endpoint, validation.Required),
		); err != nil {
			return err
		}
	}

	if c.StandName != "local" {
		if err := validation.ValidateStruct(
			c,
			validation.Field(&c.DiscoveryPostgresService, validation.Required),
			validation.Field(&c.DiscoveryRabbitService, validation.Required),
			validation.Field(&c.DiscoveryRedisService, validation.Required),
		); err != nil {
			return err
		}
	}

	return validation.ValidateStruct(
		c,
		validation.Field(&c.ServiceName, validation.Required),
		validation.Field(&c.StandName, validation.Required),
	)
}

// Validate config
func (s *Config) Validate() error {
	if err := s.GlobalConfig.Validate(); err != nil {
		return err
	}

	if err := s.LocalConfig.Validate(); err != nil {
		return err
	}

	// skip validating discovery service for the local development
	if s.LocalConfig.StandName != "local" {
		if err := s.DiscoveryConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetRabbitAddr - return random rabbit address
//
// If redis discovery addresses are empty, the address will be used from the `RABBIT_ADDR` env
func (c *Config) GetRabbitAddr() (addr string) {
	addr = c.Rabbit.Addr
	if len(c.RabbitAddrs) == 0 {
		return
	}

	addrParams := c.RabbitAddrs[utils.GetRandomInt(0, len(c.RabbitAddrs))]
	addr = fmt.Sprintf("%s:%d", addrParams.Address, addrParams.Port)

	return
}

// GetPostgresAddr - return random postgres address
//
// If postgres discovery addresses are empty, the address will be used from the `POSTGRES_ADDR` env
func (c *Config) GetPostgresAddr() (addr string) {
	addr = c.Postgres.Addr
	if len(c.PostgresAddrs) == 0 {
		return
	}

	addrParams := c.PostgresAddrs[utils.GetRandomInt(0, len(c.PostgresAddrs))]
	addr = fmt.Sprintf("%s:%d", addrParams.Address, addrParams.Port)

	return
}

// GetRedisAddr - return random redis address
//
// If redis discovery addresses are empty, the address will be used from the `REDIS_ADDR` env
func (c *Config) GetRedisAddr() (addr string) {
	addr = c.Redis.Addr
	if len(c.RedisAddrs) == 0 {
		return
	}

	addrParams := c.RedisAddrs[utils.GetRandomInt(0, len(c.RedisAddrs))]
	addr = fmt.Sprintf("%s:%d", addrParams.Address, addrParams.Port)

	return
}
