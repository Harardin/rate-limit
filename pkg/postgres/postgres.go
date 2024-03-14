package postgres

import (
	"context"
	"fmt"

	"github.com/Harardin/rate-limit/pkg/log"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Config struct {
	User   string `json:"POSTGRES_USER" secret:"true"`
	Pass   string `json:"POSTGRES_PASS" secret:"true"`
	DbName string `json:"POSTGRES_DB_NAME"`
	// For local development
	Addr string `json:"POSTGRES_ADDR"`
	// In seconds. Default 10 seconds
	PingInterval int   `json:"POSTGRES_PING_INTERVAL" default:"10"`
	MaxConns     int32 `json:"POSTGRES_MAX_CONNS"`
	MinConns     int32 `json:"POSTGRES_MIN_CONNS"`
}

func (c *Config) Validate() error {
	return validation.ValidateStruct(
		c,
		validation.Field(&c.User, validation.Required),
		validation.Field(&c.Pass, validation.Required),
		validation.Field(&c.DbName, validation.Required),
		validation.Field(&c.PingInterval, validation.Required, validation.Min(1)),
	)
}

// GetDSN return postgres dsn
//
// If addr empty, it will be used from config
func (c *Config) GetDSN(addr string) string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		c.User,
		c.Pass,
		addr,
		c.DbName,
	)
}

type PostgreSQL struct {
	*pgxpool.Pool
}

func NewPostgreSQL(ctx context.Context, logger log.Logger, cfg Config, addr string) (*PostgreSQL, error) {
	dsn := cfg.GetDSN(addr)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	config.MinConns = 3
	if cfg.MinConns != 0 {
		config.MinConns = cfg.MinConns
	}

	config.MaxConns = 6
	if cfg.MaxConns != 0 {
		config.MaxConns = cfg.MaxConns
	}

	pool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	psql := &PostgreSQL{pool}

	if err := psql.PingDB(); err != nil {
		return nil, err
	}

	logger.Info("connected to postgres")

	return psql, nil
}

func (p *PostgreSQL) PingDB() error {
	return p.Ping(context.Background())
}
