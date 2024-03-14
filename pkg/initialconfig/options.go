package initialconfig

/* Config options */

type ConfigOption func(*ConfigOptions)

type ConfigOptions struct {
	EnvPath    string
	Validation bool
}

func WithEnvPath(v string) ConfigOption {
	return func(o *ConfigOptions) {
		o.EnvPath = v
	}
}

func WithValidation(v bool) ConfigOption {
	return func(o *ConfigOptions) {
		o.Validation = v
	}
}

/* Config params options */

type ConfigParamsOption func(*ConfigParamsOptions)

type ConfigParamsOptions struct {
	ConfigType configType
}

func WithConfigType(v configType) ConfigParamsOption {
	return func(o *ConfigParamsOptions) {
		o.ConfigType = v
	}
}
