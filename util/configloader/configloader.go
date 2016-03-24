// The config package provides JSON-based configuration files, with automatic
// environment variable overriding.
package configloader

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

func New() ConfigLoader {
	return ConfigLoader{}
}

// ConfigLoader loads configuration.
type ConfigLoader struct {
	// Log is called with debug level logs about how values are resolved.
	Debug, Info, Warn func(string)
}

func (cl ConfigLoader) Load(target interface{}, filePath string) error {
	if target == nil {
		return fmt.Errorf("target was nil, need a value")
	}
	if err := cl.loadJSONFile(target, filePath); err != nil {
		return err
	}
	return cl.overrideWithEnv(target)
}

func (cl ConfigLoader) overrideWithEnv(target interface{}) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target was %T; need a struct", target)
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if err := cl.overrideField(t.Field(i), v.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func (cl ConfigLoader) overrideField(sf reflect.StructField, originalVal reflect.Value) error {
	tag := sf.Tag.Get("env")
	if tag == "" {
		return nil
	}
	envStr := os.Getenv(tag)
	if envStr == "" {
		return nil
	}
	var finalVal reflect.Value
	switch vt := originalVal.Interface().(type) {
	default:
		return fmt.Errorf("unable to override fields of type %T", originalVal.Interface())
	case string:
		finalVal = reflect.ValueOf(vt)
	case int:
		i, err := strconv.Atoi(envStr)
		if err != nil {
			return err
		}
		finalVal = reflect.ValueOf(i)
	}
	originalVal.Set(finalVal)
	return nil
}

func (cl ConfigLoader) loadJSONFile(target interface{}, filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filepath was empty")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	return json.NewDecoder(f).Decode(target)
}
