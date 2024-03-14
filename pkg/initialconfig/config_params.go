package initialconfig

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Harardin/rate-limit/pkg/utils"
)

type configType int

const (
	ConfigTypeDiscovery configType = iota + 1
	ConfigTypeGlobal
	ConfigTypeLocal
)

type envParams struct {
	IsSecret       bool
	IsJson         bool
	DiscoveryField string
	ConfigType     configType
	Value          any

	// Value from vault
	ExternalValue any
}

type Envs map[string]envParams

func GetConfigParams(item any, opts ...ConfigParamsOption) Envs {
	options := ConfigParamsOptions{}

	for _, opt := range opts {
		opt(&options)
	}

	val := reflect.ValueOf(&item).Elem()

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	envs := make(Envs, 0)

	v := reflect.ValueOf(val.Interface())
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fields := reflect.VisibleFields(v.Type())

	for _, f := range fields {
		fieldValue := v.FieldByName(f.Name).Interface()
		if f.Type.Kind() == reflect.Struct {
			cType := options.ConfigType
			if options.ConfigType == 0 {
				switch f.Name {
				case "DiscoveryConfig":
					cType = ConfigTypeDiscovery
				case "GlobalConfig":
					cType = ConfigTypeGlobal
				case "LocalConfig":
					cType = ConfigTypeLocal
				}
			}

			for name, params := range GetConfigParams(fieldValue, WithConfigType(cType)) {
				envs[name] = params
			}

			continue
		}

		if options.ConfigType == 0 {
			continue
		}

		envName := f.Tag.Get("json")
		if envName == "" || envName == "-" {
			continue
		}

		var isSecret bool
		if f.Tag.Get("secret") == "true" {
			isSecret = true
		}

		var discoveryField string
		if f.Tag.Get("discovery") != "" {
			discoveryField = f.Tag.Get("discovery")
		}

		var isJson bool
		if f.Tag.Get("is_json") == "true" {
			isJson = true
		}

		if i := strings.Index(envName, ","); i != -1 && i != 0 {
			envName = envName[:i]
		}

		envs[envName] = envParams{
			IsSecret:       isSecret,
			DiscoveryField: discoveryField,
			IsJson:         isJson,
			ConfigType:     options.ConfigType,
			Value:          fieldValue,
		}
	}

	return envs
}

func SetStructFieldValueByJsonTag(st any, e Envs, tag string, value any) error {
	if reflect.ValueOf(st).Kind() != reflect.Ptr {
		return fmt.Errorf("SetStructFieldValueByJsonTag error: struct variable must be a pointer")
	}

	if e == nil {
		return fmt.Errorf("SetStructFieldValueByJsonTag error: empty envs")
	}

	if tag == "" {
		return fmt.Errorf("SetStructFieldValueByJsonTag error: empty tag")
	}

	if value == nil {
		return fmt.Errorf("SetStructFieldValueByJsonTag error: empty value")
	}

	r := reflect.ValueOf(st)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	for i := 0; i < r.NumField(); i++ {
		f := r.Type().Field(i)
		if f.Type.Kind() == reflect.Struct {
			fieldValue := r.FieldByName(f.Name).Addr().Interface()
			if err := SetStructFieldValueByJsonTag(fieldValue, e, tag, value); err != nil {
				panic(err)
			}
			continue
		}

		envName := f.Tag.Get("json")
		if envName == "" || envName == "-" || envName != tag {
			continue
		}

		val := reflect.ValueOf(value)
		if val.Type().String() == "string" && val.Type().String() != f.Type.String() {
			valStr := value.(string)
			switch f.Type.String() {
			case "int", "int32", "int64", "uint", "uint32", "uint64", "int16", "uint16", "int8", "uint8":
				nv, err := strconv.Atoi(valStr)
				if err != nil {
					panic(err)
				}
				val = reflect.ValueOf(nv)
			case "float32", "float64":
				nv, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					panic(err)
				}
				val = reflect.ValueOf(nv)
			case "bool":
				var nv bool
				if utils.ExistInArray([]string{"1", "true"}, valStr) {
					nv = true
				}
				val = reflect.ValueOf(nv)
			case "[]string":
				nv := strings.Split(valStr, ",")
				val = reflect.ValueOf(nv)
			}
		}

		r.FieldByName(f.Name).Set(val.Convert(f.Type))
	}

	return nil
}

func GetStructFieldValueByJsonTag(st any, tag string) any {
	r := reflect.ValueOf(st)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	for i := 0; i < r.NumField(); i++ {
		f := r.Type().Field(i)

		fieldName := r.FieldByName(f.Name).Interface()
		if f.Type.Kind() == reflect.Struct {
			v := GetStructFieldValueByJsonTag(fieldName, tag)
			if v != nil {
				return v
			}
		}

		envName := f.Tag.Get("json")
		if envName == "" || envName == "-" || envName != tag {
			continue
		}

		return fieldName
	}

	return nil
}

func (s *Envs) SetValue(envName string, v any) {
	params, ok := (*s)[envName]
	if !ok {
		return
	}

	params.Value = v
	(*s)[envName] = params
}

func (s *Envs) SetExternalValue(envName string, v any) {
	params, ok := (*s)[envName]
	if !ok {
		return
	}

	params.ExternalValue = v
	(*s)[envName] = params
}

func (s *Envs) GetKeys() []string {
	res := make([]string, len(*s))
	for envName := range *s {
		res = append(res, envName)
	}

	return res
}

func (s *Envs) GetEnvsByConfigType(ct configType) Envs {
	res := make(Envs)
	for key, params := range *s {
		if params.ConfigType == ct {
			res[key] = params
		}
	}

	return res
}

func (s Envs) GetChangedEnvs(e Envs) ([]string, error) {
	if e == nil {
		return nil, fmt.Errorf("received empty envs params")
	}

	differentEnviroments := make(map[string]struct{})
	for key := range s {
		if !reflect.DeepEqual(s[key].Value, e[key].Value) ||
			!reflect.DeepEqual(s[key].ExternalValue, e[key].ExternalValue) {
			differentEnviroments[key] = struct{}{}
		}
	}

	for key := range e {
		if _, ok := s[key]; !ok {
			differentEnviroments[key] = struct{}{}
		}
	}

	differentEnviromentsArray := make([]string, 0, len(differentEnviroments))
	for key := range differentEnviroments {
		differentEnviromentsArray = append(differentEnviromentsArray, key)
	}

	return differentEnviromentsArray, nil
}
