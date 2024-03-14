package initialconfig_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Harardin/rate-limit/internal/config"
	"github.com/Harardin/rate-limit/pkg/consul"
	"github.com/Harardin/rate-limit/pkg/initialconfig"
	"github.com/Harardin/rate-limit/pkg/prometheus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructFields(t *testing.T) {
	cfg := config.Config{
		GlobalConfig: config.GlobalConfig{},
		LocalConfig: config.LocalConfig{
			ServiceName: "old name",
			Prometheus:  prometheus.Config{Port: "10001"},
			GpgPublicSignatures: map[string]string{
				"test": "test",
			},
		},
		DiscoveryConfig: config.DiscoveryConfig{
			RabbitAddrs: consul.GetServiceAddressResponse{
				{
					Address: "localhost",
					Port:    5672,
				},
			},
		},
	}

	t.Run("GetConfigParams", func(t *testing.T) {
		now := time.Now()
		res := initialconfig.GetConfigParams(cfg)
		fmt.Println("GetConfigParams ", time.Since(now))
		assert.NotNil(t, res)
		if len(res) == 0 {
			t.Fatal("empty envs map")
		}

		//spew.Dump(result)
	})

	envs := initialconfig.GetConfigParams(cfg)

	tt := []struct {
		name         string
		cfg          any
		envs         initialconfig.Envs
		tag          string
		value        any
		expected     any
		requireError bool
	}{
		{
			name:         "not pointer struct",
			cfg:          cfg,
			requireError: true,
		},
		{
			name:         "empty envs",
			cfg:          &cfg,
			requireError: true,
		},
		{
			name:         "empty tag",
			cfg:          &cfg,
			envs:         envs,
			requireError: true,
		},
		{
			name:         "empty value",
			cfg:          &cfg,
			envs:         envs,
			tag:          "SERVICE_NAME",
			requireError: true,
		},
		{
			name:  "set struct field",
			cfg:   &cfg,
			envs:  envs,
			tag:   "SERVICE_NAME",
			value: "test",
		},
		{
			name:  "set embded struct field",
			cfg:   &cfg,
			envs:  envs,
			tag:   "PROMETHEUS_PORT",
			value: "1010",
		},
		{
			name:  "set int",
			cfg:   &cfg,
			envs:  envs,
			tag:   "REDIS_DB_INDEX",
			value: 123,
		},
		{
			name:  "set int32",
			cfg:   &cfg,
			envs:  envs,
			tag:   "POSTGRES_MAX_CONNS",
			value: int32(123),
		},
		{
			name:  "set bool",
			cfg:   &cfg,
			envs:  envs,
			tag:   "RABBIT_IS_SECURE",
			value: true,
		},
		{
			name:     "set string to int",
			cfg:      &cfg,
			envs:     envs,
			tag:      "REDIS_DB_INDEX",
			value:    "1",
			expected: 1,
		},
		{
			name:     "set string to int32",
			cfg:      &cfg,
			envs:     envs,
			tag:      "POSTGRES_MAX_CONNS",
			value:    "123",
			expected: int32(123),
		},
		{
			name:     "set string to bool",
			cfg:      &cfg,
			envs:     envs,
			tag:      "RABBIT_IS_SECURE",
			value:    "true",
			expected: true,
		},
		{
			name:     "set string with int to bool",
			cfg:      &cfg,
			envs:     envs,
			tag:      "RABBIT_IS_SECURE",
			value:    "1",
			expected: true,
		},
		// {
		// 	name:     "set string to array of string",
		// 	cfg:      &cfg,
		// 	envs:     envs,
		// 	tag:      "TEST_GLOBAL",
		// 	value:    "1,2,3",
		// 	expected: []string{"1", "2", "3"},
		// },
		{
			name: "set value with custom type",
			cfg:  &cfg,
			envs: envs,
			tag:  "POSTGRES_ADDRS",
			value: consul.GetServiceAddressResponse{
				{
					Address: "localhost",
					Port:    5432,
				},
			},
		},
		{
			name: "set value with custom type",
			cfg:  &cfg,
			envs: envs,
			tag:  "RABBIT_ADDRS",
			value: consul.GetServiceAddressResponse{
				{
					Address: "localhost",
					Port:    5673,
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run("SetStructFieldValueByJsonTag_"+tc.name, func(t *testing.T) {
			oldStValue := initialconfig.GetStructFieldValueByJsonTag(tc.cfg, tc.tag)

			now := time.Now()
			err := initialconfig.SetStructFieldValueByJsonTag(tc.cfg, envs, tc.tag, tc.value)
			fmt.Println("SetStructFieldValueByJsonTag: ", time.Since(now))
			if tc.requireError {
				require.Error(t, err)
				return
			}

			newValue := initialconfig.GetStructFieldValueByJsonTag(tc.cfg, tc.tag)
			assert.NotNil(t, newValue)

			assert.NotEqual(t, tc.value, oldStValue)

			if tc.expected != nil {
				assert.Equal(t, tc.expected, newValue)
			} else {
				assert.Equal(t, tc.value, newValue)
			}

			//spew.Dump(cfg)
		})
	}

}
