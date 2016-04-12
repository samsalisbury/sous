// The config package provides JSON-based configuration files, with automatic
// environment variable overriding.
package configloader

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/opentable/sous/util/yaml"
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
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := cl.loadYAMLFile(target, filePath); err != nil {
		return err
	}
	return cl.overrideWithEnv(target)
}

func (cl ConfigLoader) overrideWithEnv(target interface{}) error {
	return cl.forEachField(target, cl.overrideField)
}

func (cl ConfigLoader) forEachField(target interface{}, f func(field reflect.StructField, val reflect.Value) error) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target was %T; need a pointer to struct", target)
	}
	v = v.Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if err := f(t.Field(i), v.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func (cl ConfigLoader) forFieldNamed(target interface{}, name string, f func(field reflect.StructField, val reflect.Value) error) error {
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

func (cl ConfigLoader) GetValue(from interface{}, name string) (interface{}, error) {
	var x interface{}
	return x, cl.forFieldNamed(from, name, func(field reflect.StructField, val reflect.Value) error {
		if field.Type.Kind() != reflect.Ptr || !val.IsNil() {
			x = val.Interface()
		}
		return nil
	})
}

func (cl ConfigLoader) SetValue(target interface{}, name, value string) error {
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

func (cl ConfigLoader) loadYAMLFile(target interface{}, filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filepath was empty")
	}
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, target)
}
