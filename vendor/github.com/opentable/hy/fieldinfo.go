package hy

import (
	"bytes"
	"reflect"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

// FieldInfo is information about a field.
type FieldInfo struct {
	Name, FieldName, PathName, KeyField, GetKeyName, SetKeyName string
	// Type is the type of this fields.
	Type,
	// KeyType is nil unless this field is a map or slice. If this field is a
	// map, then KeyType will be the type of the map's key. If it's a slice,
	// then KeyType will be int.
	KeyType,
	// ElemType is nil unless this field is a map or slice. It is the element
	// type of the map or slice.
	ElemType reflect.Type
	// GetKeyFunc is a function getting the key from this map or slice's element.
	GetKeyFunc,
	// SetKeyFunc is a function setting the key on this map or slice's element.
	SetKeyFunc reflect.Value
	// Tag is the parsed hy tag.
	Tag Tag
	// Ignore indicates this field should not be written or read by hy.
	Ignore,
	// IsField indicates this is a regular field.
	IsField,
	// IsString indicates this field should be encoded as a string.
	IsString,
	// AutoFieldName indicates this field should use a field name derived from
	// the field's name.
	AutoFieldName,
	// IsDir indicates that this map or slice field should have its elements
	// stored in a directory.
	IsDir,
	// AutoPathName indicates the file or directory storing this field should
	// have its name derived from the field's name.
	AutoPathName,
	// OmitEmpty means this field should only be written if it is not empty,
	// according to the meaning of "not empty" defined by encoding/json.
	OmitEmpty bool
}

var intType = reflect.TypeOf(1)
var strType = reflect.TypeOf("")

// NewFieldInfo creates a new FieldInfo, analysing the tag and checking the
// tag's named ID field or ID get/set methods for consistency.
func NewFieldInfo(f reflect.StructField) (*FieldInfo, error) {
	tagStr := f.Tag.Get("hy")
	tag, err := parseTag(tagStr)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid tag %q", tagStr)
	}
	jsonTag := ParseJSONTag(f)

	var fieldName, pathName, keyField,
		getKeyName, setKeyName string
	var ignore, isField, isString, autoFieldName,
		isDir, autoPathName, omitEmpty bool
	var keyType, elemType reflect.Type

	k := f.Type.Kind()

	if tag.Ignore || (tag.None && jsonTag.Ignore) {
		ignore = true
		goto done
	}

	if tag.None {
		isField = true
		if jsonTag.Name != "" {
			fieldName = jsonTag.Name
		} else {
			autoFieldName = true
		}
		omitEmpty = jsonTag.OmitEmpty
		isString = jsonTag.String
		goto done
	}

	isDir = tag.IsDir

	if tag.PathName == "." {
		autoPathName = true
	} else {
		pathName = tag.PathName
	}
	if strings.HasSuffix(tag.Key, "()") {
		getKeyName = strings.TrimSuffix(tag.Key, "()")
	} else {
		keyField = tag.Key
	}
	if strings.HasSuffix(tag.SetKey, "()") {
		setKeyName = strings.TrimSuffix(tag.SetKey, "()")
	} else {
		setKeyName = tag.SetKey
	}

	if k == reflect.Map {
		keyType = f.Type.Key()
		elemType = removePointer(f.Type.Elem())
	}
	if k == reflect.Slice {
		keyType = intType
		elemType = removePointer(f.Type.Elem())
	}

done:

	fi := &FieldInfo{
		Name:          f.Name,
		FieldName:     fieldName,
		PathName:      pathName,
		KeyField:      keyField,
		KeyType:       keyType,
		ElemType:      elemType,
		GetKeyName:    getKeyName,
		SetKeyName:    setKeyName,
		Type:          f.Type,
		Tag:           tag,
		Ignore:        ignore,
		IsField:       isField,
		IsString:      isString,
		AutoFieldName: autoFieldName,
		IsDir:         isDir,
		AutoPathName:  autoPathName,
		OmitEmpty:     omitEmpty,
	}
	return fi, errors.Wrapf(fi.Validate(),
		"analysing field %s %s %# q", f.Name, f.Type, f.Tag)
}

func removePointer(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// Validate returns any validation errors with this FieldInfo.
func (fi *FieldInfo) Validate() error {
	if err := fi.validateKeyField(); err != nil {
		return errors.Wrapf(err, "reading key field name")
	}
	//if err := validateName(fi.GetKeyName); err != nil {
	//	return errors.Wrapf(err, "reading get key method name")
	//}
	//if err := validateName(fi.SetKeyName); err != nil {
	//	return errors.Wrapf(err, "reading set key method name")
	//}
	return nil
}

func (fi *FieldInfo) validateKeyField() error {
	if fi.KeyField == "" {
		return nil
	}
	if fi.ElemType == nil {
		return nil
	}
	if err := validateName(fi.KeyField); err != nil {
		return err
	}
	if fi.ElemType.Kind() != reflect.Struct {
		return errors.Errorf("element type %s not supported; must be struct", fi.ElemType)
	}
	if fi.KeyType.Kind() != reflect.String {
		return errors.Errorf("key type %s not supported; must be string", fi.KeyType)
	}
	elemKeyField, ok := fi.ElemType.FieldByName(fi.KeyField)
	if !ok {
		return errors.Errorf("%s has no field %q", fi.ElemType, fi.KeyField)
	}
	if elemKeyField.Type != fi.KeyType {
		return errors.Errorf("%s.%s is %s; want %s (from %s)",
			fi.ElemType, elemKeyField.Name, elemKeyField.Type, fi.KeyType, fi.Type)
	}
	getFuncType := reflect.FuncOf([]reflect.Type{fi.ElemType}, []reflect.Type{fi.KeyType}, false)
	ptrToElem := reflect.PtrTo(fi.ElemType)
	setFuncType := reflect.FuncOf([]reflect.Type{ptrToElem, fi.KeyType}, nil, false)
	fi.GetKeyFunc = reflect.MakeFunc(getFuncType, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{in[0].FieldByName(fi.KeyField)}
	})
	fi.SetKeyFunc = reflect.MakeFunc(setFuncType, func(in []reflect.Value) []reflect.Value {
		elem := in[0].Elem()
		if !elem.IsValid() {
			return nil
		}
		in[0].Elem().FieldByName(fi.KeyField).Set(in[1])
		return nil
	})
	return nil
}

func validateName(s string) error {
	if len(s) == 0 {
		return nil
	}
	rs := bytes.Runes([]byte(s))
	if s == "_" || (!unicode.IsLetter(rs[0]) && rs[0] != '_') {
		return errors.Errorf("illegal token %q", s)
	}
	for _, r := range rs[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return errors.Errorf("illegal token %q", s)
		}
	}
	return nil
}

// Following code copied from https://golang.org/src/encoding/json

// tagOptions is the string following a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseJSONTagOptions(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}

// JSONTag represents a json field tag.
type JSONTag struct {
	Ignore, String, OmitEmpty bool
	Name                      string
}

// ParseJSONTag parses a json field tag from a struct field.
// Only strings, floats, integers, and booleans can be quoted.
func ParseJSONTag(field reflect.StructField) JSONTag {
	jsonTag := field.Tag.Get("json")
	var ignore, str, omitEmpty bool
	name, opts := parseJSONTagOptions(jsonTag)
	if opts.Contains("string") {
		switch field.Type.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String:
			str = true
		}
	}
	if opts.Contains("omitempty") {
		omitEmpty = true
	}
	if jsonTag == "-" {
		ignore = true
	}
	return JSONTag{Name: name, Ignore: ignore, String: str, OmitEmpty: omitEmpty}
}
