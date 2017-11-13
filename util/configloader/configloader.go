// Package configloader provides YAML-based configuration files, with
// automatic environment variable overriding.
package configloader

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/yaml"
)

// New returns a new ConfigLoader.
func New() ConfigLoader {
	return &configLoader{}
}

type (
	// ConfigLoader loads configuration.
	ConfigLoader interface {
		//Loads a YAML formated configuration from path into data.
		Load(data interface{}, path string) error
		// SetValue sets a value at path key, as long as it can convert value to
		// the correct type for that key. Otherwise it returns an error.
		SetValue(target interface{}, key, value string) error
		// SetValidValue is similar to SetValue except is also returns an error
		// if the resulting target is not valid. (If target does not implement
		// Validator then it is assumed to be valid.)
		SetValidValue(target interface{}, key, value string) error
		// GetValue returns the value of key in target, or nil and a non-nil
		// error if that key is not found.
		GetValue(target interface{}, key string) (interface{}, error)
	}

	// DefaultFiller can fill defaults for a config
	DefaultFiller interface {
		FillDefaults() error
	}

	// A Validator validates
	Validator interface {
		Validate() error
	}

	configLoader struct {
		// Log is called with debug level logs about how values are resolved.
	}
)

func (cl *configLoader) Load(target interface{}, filePath string) error {
	if target == nil {
		return fmt.Errorf("target was nil, need a value")
	}
	_, err := os.Stat(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// I'd like to say "override with e.g. SOUS_CONFIG_FILE", but the
		// construction of the path happens elsewhere.
		logging.Log.Info.Printf("No config file found at %q, using defaults.", filePath)
	} else {
		if err := cl.loadYAMLFile(target, filePath); err != nil {
			return err
		}
	}
	if fd, ok := target.(DefaultFiller); ok {
		if err := fd.FillDefaults(); err != nil {
			return err
		}
	}
	return cl.overrideWithEnv(target)
}

func (cl *configLoader) Validate(target interface{}) error {
	if validator, ok := target.(Validator); ok {
		return validator.Validate()
	}
	return nil
}

func (cl *configLoader) overrideWithEnv(target interface{}) error {
	return cl.forEachField(target, cl.overrideField)
}

func (cl *configLoader) forEachField(target interface{}, f func(field reflect.StructField, val reflect.Value) error) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target was %T; need a pointer to struct", target)
	}
	v = v.Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		switch field.Type.Kind() {
		default:
			if err := f(field, value); err != nil {
				return err
			}
		case reflect.Struct:
			if err := cl.forEachField(value.Addr().Interface(), f); err != nil {
				return err
			}
		case reflect.Ptr:
			if field.Type.Elem().Kind() == reflect.Struct {
				if err := cl.forEachField(value.Interface(), f); err != nil {
					return err
				}
				continue
			}

			if err := f(field, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (cl *configLoader) forFieldNamed(target interface{}, name string, f func(field reflect.StructField, val reflect.Value) error) error {
	found := false
	err := cl.forEachField(target, func(field reflect.StructField, val reflect.Value) error {
		if strings.ToLower(field.Name) == strings.ToLower(name) {
			found = true
			return f(field, val)
		}
		return nil
	})
	if !found {
		return fmt.Errorf("config value %s does not exist", name)
	}
	return err
}

func (cl *configLoader) GetValue(from interface{}, name string) (interface{}, error) {
	var x interface{}
	return x, cl.forFieldNamed(from, name, func(field reflect.StructField, val reflect.Value) error {
		if field.Type.Kind() != reflect.Ptr || !val.IsNil() {
			x = val.Interface()
		}
		return nil
	})
}

func (cl *configLoader) SetValue(target interface{}, name, value string) error {
	return cl.forFieldNamed(target, name, func(field reflect.StructField, val reflect.Value) error {
		switch k := field.Type.Kind(); k {
		default:
			return fmt.Errorf("configloader does not know how to set fields of kind %s", k)
		case reflect.String:
			val.Set(reflect.ValueOf(value))
		case reflect.Int:
			v, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			val.Set(reflect.ValueOf(v))
		}
		return nil
	})
}

func (cl *configLoader) SetValidValue(target interface{}, name, value string) error {
	if err := cl.SetValue(target, name, value); err != nil {
		return err
	}
	return cl.Validate(target)
}

func (cl *configLoader) overrideField(sf reflect.StructField, originalVal reflect.Value) error {
	envName := sf.Tag.Get("env")
	if envName == "" {
		return nil
	}
	envVal, present := os.LookupEnv(envName)
	if !present {
		return nil
	}
	logging.Log.Debug.Printf("Environment configuration OVERRIDE: %s=%s\n", envName, envVal)
	var finalVal reflect.Value
	switch originalVal.Interface().(type) {
	default:
		return fmt.Errorf("unable to override fields of type %T", originalVal.Interface())
	case string:
		finalVal = reflect.ValueOf(envVal)
	case int:
		i, err := strconv.Atoi(envVal)
		if err != nil {
			return err
		}
		finalVal = reflect.ValueOf(i)
	}
	originalVal.Set(finalVal)
	return nil
}

func (cl *configLoader) loadYAMLFile(target interface{}, filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filepath was empty")
	}
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, target)
}
