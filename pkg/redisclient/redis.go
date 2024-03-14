package redisclient

import validation "github.com/go-ozzo/ozzo-validation/v4"

type Config struct {
	Addr         string `json:"REDIS_ADDR"`
	User         string `json:"REDIS_USER" secret:"true"`
	Pass         string `json:"REDIS_PASS" secret:"true"`
	DbIndex      int    `json:"REDIS_DB_INDEX"`
	PingInterval int    `json:"REDIS_PING_INTERVAL" default:"10"`
}

func (c *Config) Validate() error {
	return validation.ValidateStruct(
		c,
		validation.Field(&c.PingInterval, validation.Required),
	)
}
