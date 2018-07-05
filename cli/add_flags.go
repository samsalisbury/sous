package cli

import (
	"flag"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// MustAddFlags wraps AddFlags and panics if AddFlags returns an error.
func MustAddFlags(fs *flag.FlagSet, target interface{}, help string, optDefaults ...map[string]interface{}) {
	if err := AddFlags(fs, target, help, optDefaults...); err != nil {
		panic(err)
	}
}

// AddFlags sniffs out struct fields from target and adds them as var flags to
// the flag set.
func AddFlags(fs *flag.FlagSet, target interface{}, help string, optDefaults ...map[string]interface{}) error {
	defaults := map[string]interface{}{}
	if len(optDefaults) > 0 {
		defaults = optDefaults[0]
	}

	if fs == nil {
		return errors.Errorf("cannot add flags to nil *flag.FlagSet")
	}
	v := reflect.ValueOf(target)
	k := v.Kind()
	if target == nil || k != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.Errorf("target is %T; want pointer to struct", target)
	}

	usage, err := parseUsage(help)
	if err != nil {
		return errors.Wrapf(err, "parsing usage text")
	}

	v = v.Elem()
	t := v.Type()
	numFields := t.NumField()
	for i := 0; i < numFields; i++ {
		f := t.Field(i)
		ft := f.Type
		fp := v.Field(i).Addr().Interface()
		if f.Type.Kind() == reflect.Struct {
			if err := AddFlags(fs, fp, help, optDefaults...); err != nil {
				return err
			}
		}
		name := strings.ToLower(f.Name)
		if tag := f.Tag.Get("flag"); tag != "" {
			name = tag
		}
		u, ok := usage[name]
		if !ok {
			continue
		}
		switch field := fp.(type) {
		default:
			return errors.Errorf("target field %s.%s is %s; want string, int, or bool", t, f.Name, ft)
		case *string:
			fs.StringVar(field, name, getDefault(name, "", defaults).(string), u)
		case *bool:
			fs.BoolVar(field, name, getDefault(name, false, defaults).(bool), u)
		case *int:
			fs.IntVar(field, name, getDefault(name, 0, defaults).(int), u)
		}
	}

	return nil
}

func getDefault(name string, ifMissing interface{}, defs map[string]interface{}) interface{} {
	if val, ok := defs[name]; ok {
		return val
	}
	return ifMissing
}

func parseUsage(s string) (map[string]string, error) {
	parts := strings.Split("\t"+strings.TrimSpace(s), "\t-")
	m := make(map[string]string, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) == 0 {
			continue
		}
		lines := strings.Split(p, "\n")
		if len(lines) < 2 {
			return nil, errors.Errorf("section has too few lines: %q", p)
		}
		name := strings.Split(lines[0], " ")[0]
		usage := strings.TrimSpace(lines[1])
		m[name] = usage
	}
	return m, nil
}
