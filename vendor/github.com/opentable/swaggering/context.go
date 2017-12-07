package swaggering

import (
	"fmt"
)

type Context struct {
	packageName, importName string
	swaggers                []*Swagger
	openModels              []*Model

	codefiles []*CodeFile
	structs   map[string]*Struct
}

func NewContext(packageName, importName string) (context *Context) {
	context = &Context{packageName: packageName, importName: importName}
	context.swaggers = make([]*Swagger, 0)

	return
}

func (context *Context) Resolve() {

	/*
		for _, swagger := range context.swaggers {
			for adx := range swagger.Apis {
				context.apis = append(context.apis, &swagger.Apis[adx])
			}
			for _, model := range swagger.Models {
				context.models = append(context.models, model)
			}
		}

		log.Printf("  Found %d apis", len(context.apis))
		log.Printf("  Found %d models", len(context.models))

		context.openModels = make([]*Model, 0, len(context.models))
	*/

	context.resolveApis()
	context.resolveModels()
}

func (context *Context) resolveApis() {
	for _, swagger := range context.swaggers {
		file := CodeFile{}
		context.codefiles = append(context.codefiles, &file)

		file.BasePackageName = context.packageName
		file.PackageImportName = context.importName
		file.Name = swagger.name

		for _, api := range swagger.Apis {

			for _, op := range api.Operations {
				method := context.resolveOperation(op)
				method.Path = api.Path

				file.Methods = append(file.Methods, method)
			}
		}
	}
}

func (context *Context) resolveOperation(op *Operation) *Method {
	method := Method{hostPackage: context.packageName}

	method.Name = capitalize(op.Nickname)
	method.Method = op.Method

	mtype, err := op.findGoType(context)
	if err != nil {
		logErr(err, "Operation %s invalid: %v", op.Nickname)
		method.invalidity = true
	}

	if mtype != nil {
		method.DTORequest = !isPrimitive(mtype)
		method.Results = append(method.Results, &Field{Name: "response", TypeStringer: mtype})
	}

	for _, parm := range op.Parameters {
		field := Field{Name: parm.Name}
		prm := Param{Field: &field, ParamType: parm.ParamType}
		method.Params = append(method.Params, &prm)

		t, err := parm.findGoType(context)
		logErr(err, "Operation %s invalid: parameter %s: %v", op.Nickname, parm.Name)

		field.TypeStringer = t

		if !t.Valid() {
			method.invalidity = true
		}

		if parm.Name == "body" {
			method.HasBody = true
		}
	}
	return &method
}

func (context *Context) modelFor(typeName string) (TypeStringer, error) {
	t, err := context.modelUsed(typeName)
	if err != nil {
		return &Pointer{
			&Struct{
				invalidity: true,
				Name:       typeName,
				Package:    "notfound",
			},
		}, err
	}

	return &Pointer{t}, nil
}

func (context *Context) modelUsed(name string) (TypeStringer, error) {
	for _, swagger := range context.swaggers {
		for _, model := range swagger.Models { // XXX it's a map - simply use Models[name] ?
			if model.Id == name {
				if !model.resolved {
					context.openModels = append(context.openModels, model)
				}

				return context.getStruct(name)
			}
		}
	}
	return nil, fmt.Errorf("model %q doesn't appear in known models", name)
}

func (context *Context) getStruct(name string) (*Struct, error) {
	if context.structs == nil {
		context.structs = map[string]*Struct{}
	}
	if s, has := context.structs[name]; has {
		return s, nil
	}
	context.structs[name] = &Struct{
		Package: "dtos",
		Name:    name,
	}
	return context.structs[name], nil
}

func (context *Context) resolveModels() {
	var cur *Model

	for len(context.openModels) > 0 {
		cur, context.openModels = context.openModels[0], context.openModels[1:]
		if cur.resolved {
			continue
		}
		context.resolveModel(cur)
	}
}

func (context *Context) resolveModel(model *Model) *Struct {
	model.resolved = true

	s, err := context.getStruct(model.Id)
	logErr(err, "when getting struct by name: %q: %v", model.Id)

	for name, prop := range model.Properties {
		field, err := context.resolveProperty(name, prop)
		logErr(err, "when resolving property type for %q: %v", model.Id)

		if field == nil {
			continue
		}
		attr := Attribute{
			Field:       field,
			SwaggerName: name,
		}
		s.Fields = append(s.Fields, &attr)

		switch enum := field.TypeStringer.(type) {
		case nil:
		case *EnumType:
			enum.HostModel = model.Id

			exists := false
			for _, e := range s.Enums {
				if e.Name == enum.Name {
					exists = true
					break
				}
			}

			if !exists {
				s.Enums = append(s.Enums, enum)
			}
		}
	}

	return s
}

func (context *Context) resolveProperty(name string, prop *Property) (*Field, error) {
	t, err := prop.findGoType(context) //uses embedded Collection's impl
	return &Field{Name: capitalize(name), TypeStringer: t}, err
}
