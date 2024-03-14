package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/Harardin/rate-limit/pkg/log"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

type Vault interface {
	GetSecret(ctx context.Context, path string) (interface{}, error)
	GetSecretByKey(ctx context.Context, path, key string) (interface{}, error)
}

type service struct {
	logger     log.Logger
	vaultToken string
	client     *api.Client
	kv         *api.KVv2
}

func NewVault(logger log.Logger, vaultAddr string, vaultKey string, mountPath string) (Vault, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}
	client.SetAddress(vaultAddr)
	client.SetToken(vaultKey)

	s := &service{
		logger,
		vaultKey,
		client,
		client.KVv2(mountPath),
	}

	go s.renewToken()

	return s, nil
}

func (s *service) renewToken() {
	for {
		apiSecret, err := s.client.Auth().Token().LookupSelf()
		if err != nil {
			s.logger.Fatalf("get token info from vault error: %v", err)
		}

		tData, err := getTokenData(apiSecret.Data)
		if err != nil {
			s.logger.Fatalf("failed to get token data: %v", err)
		}

		// If root token, stop renew
		if tData.isRoot {
			s.logger.Info("vault token is root. stop renew token")
			return
		}

		if !tData.isRenewable {
			s.logger.Fatal("vault token is not renewable")
		}

		timeLeft := time.Until(tData.expirationTime)

		if _, err := s.client.Auth().Token().RenewSelf(60 * 60 * 8); err != nil {
			s.logger.Errorf("renew vault token error: %v", err)

			if timeLeft <= time.Minute {
				s.logger.Fatal("failed to renew token")
			}
		}

		s.logger.Debug("vault token was updated")

		// Sleep 10 minutes, before next update
		time.Sleep(time.Minute * 10)
	}
}

func getTokenData(data map[string]interface{}) (*tokenData, error) {
	if data == nil {
		return nil, fmt.Errorf("vault token data is nil")
	}

	displayName, err := getStringFromMap(data, "display_name")
	if err != nil {
		return nil, errors.Wrap(err, "getStringFromMap error")
	}

	expTimeI, err := getIntrefaceFromMap(data, "expire_time")
	if err != nil {
		return nil, err
	}

	if displayName == "root" || expTimeI == nil {
		return &tokenData{
			isRoot: true,
		}, nil
	}

	isRenewable, err := getBoolFromMap(data, "renewable")
	if err != nil {
		return nil, errors.Wrap(err, "getBoolFromMap error")
	}

	expirationTime, err := getStringFromMap(data, "expire_time")
	if err != nil {
		return nil, errors.Wrap(err, "getStringFromMap error")
	}

	expTime, err := time.Parse(time.RFC3339, expirationTime)
	if err != nil {
		return nil, errors.Wrap(err, "expirationTime parse error")
	}

	return &tokenData{
		isRenewable:    isRenewable,
		expirationTime: expTime,
	}, nil
}

func getIntrefaceFromMap(m map[string]any, key string) (any, error) {
	valI, ok := m[key]
	if !ok {
		return "", fmt.Errorf("map does not exist key \"%s\"", key)
	}

	return valI, nil
}

func getBoolFromMap(m map[string]any, key string) (bool, error) {
	valI, err := getIntrefaceFromMap(m, key)
	if err != nil {
		return false, err
	}

	val, ok := valI.(bool)
	if !ok {
		return false, fmt.Errorf("value \"%v\" does not implement bool type", valI)
	}

	return val, nil
}

func getStringFromMap(m map[string]any, key string) (string, error) {
	valI, err := getIntrefaceFromMap(m, key)
	if err != nil {
		return "", err
	}

	val, ok := valI.(string)
	if !ok {
		return "", fmt.Errorf("value \"%v\" does not implement string type", valI)
	}

	return val, nil
}

func (s *service) GetSecret(ctx context.Context, path string) (interface{}, error) {
	secret, err := s.kv.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	value, ok := secret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("field \"value\" does not exist in vault path \"%s\"", path)
	}

	return value, nil
}

func (s *service) GetSecretByKey(ctx context.Context, path, key string) (interface{}, error) {
	secret, err := s.kv.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	value, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("key \"%s\" does not exist in vault", key)
	}

	return value, nil
}
