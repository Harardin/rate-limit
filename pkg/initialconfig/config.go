package initialconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/Harardin/rate-limit/internal/config"
	"github.com/Harardin/rate-limit/pkg/consul"
	"github.com/Harardin/rate-limit/pkg/log"
	"github.com/Harardin/rate-limit/pkg/utils"
	"github.com/Harardin/rate-limit/pkg/vault"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigdotenv"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type IConfig interface {
	Validate() error
}

type cfgService struct {
	logger log.Logger

	initialConfig *initialConfig
	mainConfig    *config.Config

	mu   sync.RWMutex
	envs Envs

	consulClient consul.Consul
	vaultClient  vault.Vault
}

type initialConfig struct {
	// Vault Config
	VaultEnabled      bool   `json:"VAULT_ENABLED" default:"true"`
	VaultGeneralUrl   string `json:"VAULT_GENERAL_URL"`
	VaultGeneralToken string `json:"VAULT_GENERAL_TOKEN"`
	VaultMountPath    string `json:"VAULT_MOUNT_PATH"`

	// Consul Config
	ConsulEnabled      bool   `json:"CONSUL_ENABLED" default:"true"`
	ConsulGeneralUrl   string `json:"CONSUL_GENERAL_URL"`
	ConsulGeneralToken string `json:"CONSUL_GENERAL_TOKEN"`
}

func (c *initialConfig) Validate() error {
	// Validate consul
	if c.ConsulEnabled {
		if err := validation.ValidateStruct(
			c,
			validation.Field(&c.ConsulGeneralUrl, validation.Required),
		); err != nil {
			return err
		}
	}

	// Validate vault
	if c.VaultEnabled {
		if err := validation.ValidateStruct(
			c,
			validation.Field(&c.VaultGeneralUrl, validation.Required),
			validation.Field(&c.VaultGeneralToken, validation.Required),
			validation.Field(&c.VaultMountPath, validation.Required),
		); err != nil {
			return err
		}
	}

	return nil
}

// LoadConfig accepts logger to track on config change
func LoadConfig(l log.Logger, mainConfig *config.Config) chan []string {
	// Load initial config
	initConfig := new(initialConfig)
	if err := LoadConfigFromEnv(initConfig); err != nil {
		l.Fatalf("failed to load initial config: %v", err)
	}

	// Load local config
	if err := LoadConfigFromEnv(mainConfig, WithValidation(false)); err != nil {
		l.Fatalf("failed to load local config: %v", err)
	}

	cs := cfgService{
		logger:        l,
		initialConfig: initConfig,
		mainConfig:    mainConfig,
	}

	// Connect to consul
	var err error
	if initConfig.ConsulEnabled {
		cs.consulClient, err = consul.NewConsul(mainConfig.ServiceName, mainConfig.StandName, initConfig.ConsulGeneralUrl, initConfig.ConsulGeneralToken)
		if err != nil {
			l.Fatalf("failed to init consul instance: %v", err)
		}
		l.Info("connected to consul")
	}

	// Connect to vault
	if initConfig.VaultEnabled {
		cs.vaultClient, err = vault.NewVault(l, initConfig.VaultGeneralUrl, initConfig.VaultGeneralToken, initConfig.VaultMountPath)
		if err != nil {
			l.Fatalf("failed to init vault instance: %v", err)
		}
		l.Info("connected to vault")
	}

	if initConfig.ConsulEnabled && initConfig.VaultEnabled {
		envs, err := cs.getDataFromConsulAndVault()
		if err != nil {
			l.Fatalf("getDataFromConsulAndVault error: %v", err)
		}

		cs.mu.Lock()
		cs.envs = envs
		cs.mu.Unlock()

		for envName, params := range cs.envs {
			value := params.Value
			if params.IsSecret {
				value = params.ExternalValue
			}

			if value == nil {
				continue
			}

			if err := SetStructFieldValueByJsonTag(mainConfig, cs.envs, envName, value); err != nil {
				l.Fatalf("failed to set config env \"%s\": %v", envName, err)
			}
		}
	}

	// Validate main config
	if err := mainConfig.Validate(); err != nil {
		l.Fatalf("failed to validate local config: %v", err)
	}

	c := make(chan []string, 1)
	if !initConfig.ConsulEnabled || !initConfig.VaultEnabled {
		return c
	}

	// Watch consul for changes
	go func(c chan []string) {
		for {
			randomSleepSeconds := utils.GetRandomInt(150, 250)
			time.Sleep(time.Second * time.Duration(randomSleepSeconds))

			newEnvs, err := cs.getDataFromConsulAndVault()
			if err != nil {
				l.Fatalf("getDataFromConsulAndVault error: %v", err)
			}

			changedEnvs, err := cs.envs.GetChangedEnvs(newEnvs)
			if err != nil {
				l.Errorf("failed to check is equal envs: %v", err)
				continue
			}

			if len(changedEnvs) == 0 {
				continue
			}

			cs.mu.Lock()
			cs.envs = newEnvs
			cs.mu.Unlock()

			for envName, params := range cs.envs {
				if !utils.ExistInArray(changedEnvs, envName) {
					continue
				}

				value := params.Value
				if params.IsSecret {
					value = params.ExternalValue
				}

				if value == nil {
					continue
				}

				if err := SetStructFieldValueByJsonTag(mainConfig, cs.envs, envName, value); err != nil {
					l.Fatalf("failed to set config env \"%s\": %v", envName, err)
				}
			}

			l.Info("main config was updated")

			c <- changedEnvs
		}
	}(c)

	return c
}

// LoadConfigFromEnv - load environment variables from `os env`, `.env` file and pass it to struct.
//
// For local development use `.env` file from root project.
//
// LoadConfigFromEnv also call a `Validate` method.
//
// Example:
//
//	cfg := new(config.Config)
//	if err := initialconfig.LoadConfigFromEnv(cfg); err != nil {
//		log.Fatalf("could not load configuration: %v", err)
//	}
func LoadConfigFromEnv(cfg IConfig, opts ...ConfigOption) error {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return fmt.Errorf("config variable must be a pointer")
	}

	options := ConfigOptions{
		Validation: true,
	}

	for _, opt := range opts {
		opt(&options)
	}

	if options.EnvPath == "" {
		pwdDir, err := os.Getwd()
		if err != nil {
			return err
		}
		options.EnvPath = pwdDir
	}

	aconf := aconfig.Config{
		AllowUnknownFields: true,
		SkipFlags:          true,
		Files:              []string{path.Join(options.EnvPath, ".env")},
		FileDecoders: map[string]aconfig.FileDecoder{
			".env": aconfigdotenv.New(),
		},
	}

	loader := aconfig.LoaderFor(cfg, aconf)
	if err := loader.Load(); err != nil {
		return err
	}

	if !options.Validation {
		return nil
	}

	return cfg.Validate()
}

func (s *cfgService) getDataFromConsulAndVault() (Envs, error) {
	if s.consulClient == nil {
		return nil, fmt.Errorf("empty consul client")
	}

	if s.vaultClient == nil {
		return nil, fmt.Errorf("empty vault client")
	}

	result := GetConfigParams(*s.mainConfig)

	for envName, params := range result {
		path := "local"
		if params.ConfigType == ConfigTypeGlobal {
			path = "global"
		}

		res, err := s.consulClient.GetValue(context.Background(), path, envName)
		if err != nil && !errors.Is(err, consul.ErrKeyNotExist) {
			return nil, fmt.Errorf("failed to get data from consul: %v", err)
		}

		// TODO: implemented in future
		if params.IsJson {
			continue
		}

		consulValue := string(res)
		if consulValue == "" {
			continue
		}

		if params.ConfigType == ConfigTypeDiscovery {
			s.logger.Errorf("bad env \"%s\". you can't set service discovery addrs from consul. please, delete enviroment \"%s\" from consul kv storage", envName, envName)
			continue
		}

		result.SetValue(envName, consulValue)

		// Get data from service discovery
		if params.DiscoveryField != "" {
			res, err := s.consulClient.GetServiceAddress(context.Background(), consulValue)
			if err != nil {
				return nil, fmt.Errorf("failed to get data from consul: %v", err)
			}

			if s.mainConfig.StandName != "local" && len(res) == 0 {
				s.logger.Errorf("consul discovery return empty response for consul service \"%s\". env \"%s\" will be empty", consulValue, params.DiscoveryField)
			}

			result.SetValue(params.DiscoveryField, res)
		}

		// Get data from vault
		if params.IsSecret {
			value, err := s.vaultClient.GetSecret(context.Background(), consulValue)
			if err != nil {
				return nil, fmt.Errorf("failed to get secret from vault: %v", err)
			}

			result.SetExternalValue(envName, value)
		}
	}

	return result, nil
}
