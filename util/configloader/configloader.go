// The configloader package provides YAML-based configuration files, with
// automatic environment variable overriding.
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

func New() *configLoader {
	return &configLoader{}
}

type (
	// ConfigLoader loads configuration.
	ConfigLoader interface {
		//Loads a YAML formated configuration from path into data.
		Load(data interface{}, path string) error
	}

	DefaultFiller interface {
		FillDefaults() error
	}
	Validator interface {
		Validate() error
	}

	configLoader struct {
		// Log is called with debug level logs about how values are resolved.
		Debug, Info func(...interface{})
	}
)

// SetLogFunc implements sous.ILogger on configLoader
func (cl *configLoader) SetLogFunc(f func(...interface{})) {
	cl.Info = f
}

// SetDebugFunc implements sous.ILogger on configLoader
func (cl *configLoader) SetDebugFunc(f func(...interface{})) {
	cl.Debug = f
}

func (cl *configLoader) Load(target interface{}, filePath string) error {
	if target == nil {
		return fmt.Errorf("target was nil, need a value")
	}
	_, err := os.Stat(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		//cl.Info("Missing config file, using defaults", map[string]interface{}{"path": filePath})
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
	if err := cl.overrideWithEnv(target); err != nil {
		return err
	}
	return nil
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

func (cl *configLoader) overrideField(sf reflect.StructField, originalVal reflect.Value) error {
	envName := sf.Tag.Get("env")
	if envName == "" {
		return nil
	}
	envVal, present := os.LookupEnv(envName)
	if !present {
		return nil
	}
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
