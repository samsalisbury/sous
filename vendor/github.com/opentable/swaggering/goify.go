package swaggering

import (
	"fmt"
	"regexp"
	"strings"
)

func capitalize(word string) string {
	firstRE := regexp.MustCompile(`^.`)
	return firstRE.ReplaceAllStringFunc(word, func(match string) string {
		return strings.ToTitle(match)
	})
}

func goName(name string) string {
	if name == "Object" {
		return "interface{}"
	}
	return name
}

var listRE = regexp.MustCompile(`^List\[([^,]*)]`)
var mapRE = regexp.MustCompile(`^Map\[([^,]*),(.*)]`)

// c.f. the embeded relationship of Parameter -> SwaggerType
func (p *Parameter) findGoType(context *Context) (TypeStringer, error) {
	if p.ParamType == "body" {
		return context.modelFor(p.Type)
	}
	return findGoType(context, &p.SwaggerType)
}

// c.f. the embeded relationship of Collection -> SwaggerType
func (c *Collection) findGoType(context *Context) (TypeStringer, error) {
	if c.Type == "array" {
		t, err := findGoType(context, &c.Items)
		if err != nil {
			return nil, err
		}

		switch item := t.(type) {
		default:
			return &SliceType{items: item}, nil
		case *Pointer:
			return &SliceType{items: item.TypeStringer}, nil
		}
	}
	return findGoType(context, &c.SwaggerType)
}

func (op *Operation) findGoType(context *Context) (TypeStringer, error) {
	if op.Type == "" {
		op.Type = "array"
	}
	return op.Collection.findGoType(context)
}

func isAggregate(kind string) bool {
	return mapRE.FindStringSubmatch(kind) != nil || listRE.FindStringSubmatch(kind) != nil
}

func findGoType(context *Context, from *SwaggerType) (TypeStringer, error) {
	switch {
	default:
		return primitiveOrRefType(context, from)
	case len(from.Enum) > 0:
		return &EnumType{
			Name:   from.Ref,
			Values: from.Enum,
		}, nil
	case from.Type == "":
		return refType(context, from.Ref)
	}
}

func primitiveOrRefType(context *Context, from *SwaggerType) (TypeStringer, error) {
	t, err := goPrimitiveFormattedType(from.Type, from.Format)
	if err != nil {
		return refType(context, from.Type)
	}
	return t, nil
}

func refType(context *Context, refStr string) (TypeStringer, error) {
	t, err := aggregateType(context, refStr)
	if err != nil {
		return context.modelFor(refStr)
	}
	return t, err
}

func aggregateItemType(context *Context, typeStr string) (TypeStringer, error) {
	if typeStr == "Object" {
		return &PrimitiveType{Name: "interface{}"}, nil
	}
	t, err := aggregateType(context, typeStr)
	if err != nil {
		return goPrimitiveOrModel(context, typeStr)
	}
	return t, err
}

func aggregateType(context *Context, typeStr string) (TypeStringer, error) {
	if matches := mapRE.FindStringSubmatch(typeStr); matches != nil {
		keys, err := goPrimitiveType(matches[1])
		if err != nil {
			return nil, err
		}

		values, err := aggregateItemType(context, matches[2])
		if err != nil {
			return nil, err
		}

		return &MapType{
			keys:   keys,
			values: values,
		}, nil

	}

	if matches := listRE.FindStringSubmatch(typeStr); matches != nil {
		values, err := aggregateItemType(context, matches[1])
		if err != nil {
			return nil, err
		}

		return &SliceType{items: values}, nil
	}

	return nil, fmt.Errorf("Not recognized as an aggregate type: %s", typeStr)
}

func goPrimitiveOrModel(context *Context, name string) (TypeStringer, error) {
	t, err := goPrimitiveType(name)
	if err != nil {
		return context.modelUsed(name)
	}
	return t, nil
}

func goPrimitiveType(sType string) (TypeStringer, error) {
	return goPrimitiveFormattedType(sType, "")
}

func goPrimitiveFormattedType(sType, format string) (TypeStringer, error) {
	var t string
	switch sType {
	default:
		return nil, fmt.Errorf("Unrecognized primitive type: %s", sType)
	case "boolean":
		t = "bool"
	case "integer", "int":
		switch format {
		default:
			t = "int64"
		case "int32":
			t = "int32"
		}
	case "number":
		switch format {
		case "float", "none":
			t = "float32"
		default:
			t = "float64"
		}
	case "string":
		switch format {
		case "date", "data-time":
			t = "time.Time"
		default:
			t = "string"
		}
	}
	return &PrimitiveType{Name: t}, nil
}
