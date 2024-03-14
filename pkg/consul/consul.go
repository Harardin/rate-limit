package consul

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
)

type Consul interface {
	GetValue(ctx context.Context, path, key string) ([]byte, error)
	GetServiceAddress(ctx context.Context, serviceName string) (GetServiceAddressResponse, error)
}

type service struct {
	serviceName string
	standName   string
	client      *api.Client
	kv          *api.KV
	health      *api.Health
}

func NewConsul(serviceName, standName, consulAddr, consulToken string) (Consul, error) {
	config := api.DefaultConfig()
	config.Address = consulAddr
	config.Token = consulToken

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &service{
		serviceName,
		standName,
		client,
		client.KV(),
		client.Health(),
	}, nil
}

func (s *service) GetValue(ctx context.Context, path, key string) ([]byte, error) {
	if path == "local" {
		path = path + "/" + s.serviceName
	}

	fullPath := fmt.Sprintf("%s/%s/%s", s.standName, path, key)
	pair, _, err := s.kv.Get(fullPath, nil)
	if err != nil {
		return nil, err
	}

	if pair == nil {
		return nil, ErrKeyNotExist
	}

	return pair.Value, nil
}

func (s *service) GetServiceAddress(ctx context.Context, serviceName string) (GetServiceAddressResponse, error) {
	r, _, err := s.health.Service(serviceName, "", true, nil)
	if err != nil {
		return nil, err
	}

	// if len(r) == 0 {
	// 	return nil, fmt.Errorf("empty consul response")
	// }

	res := make(GetServiceAddressResponse, 0)
	for _, item := range r {
		for _, ta := range item.Service.TaggedAddresses {
			var exist bool
			for _, existItem := range res {
				if existItem.Address == ta.Address && existItem.Port == ta.Port {
					exist = true
					break
				}
			}

			if !exist {
				res = append(res, GetServiceAddressResponseItem{
					Address: ta.Address,
					Port:    ta.Port,
				})
			}
		}
	}

	return res, nil
}
