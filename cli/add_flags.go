package cli

import (
	"flag"
	"log"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// AddFlags sniffs out struct fields from target and adds them as var flags to
// the flag set.
func AddFlags(fs *flag.FlagSet, target interface{}, help string) error {
	if fs == nil {
		return errors.Errorf("cannot add flags to nil *flag.FlagSet")
	}
	v := reflect.ValueOf(target)
	k := v.Kind()
	if target == nil || k != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.Errorf("target is %T; want pointer to struct", target)
	}

	v = v.Elem()
	t := v.Type()
	numFields := t.NumField()
	for i := 0; i < numFields; i++ {
		f := t.Field(i)
		ft := f.Type
		fp := v.Field(i).Addr().Interface()
		name := strings.ToLower(f.Name)
		switch field := fp.(type) {
		default:
			return errors.Errorf("target field %s.%s is %s; want string, int",
				t, f.Name, ft)
		case *string:
			log.Println("Adding field", name)
			fs.StringVar(field, name, "", "usage")
		}
	}

	return nil
}
