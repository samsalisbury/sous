package yaml

import (
	"bytes"
	"fmt"
	"text/template"
)

type Template struct {
	Template *template.Template
}

type TemplateWithPipeline struct {
	Template *template.Template
	Pipeline interface{}
}

type RenderedYAML struct {
	Bytes []byte
	Error error
}

func (y *RenderedYAML) String() string {
	if y.Error != nil {
		return fmt.Sprintf("ERROR: %s", y.Error)
	}
	return string(y.Bytes)
}

func NewTemplateFromValue(v interface{}) (*Template, error) {
	s, err := Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("Unable to marshal original value to YAML: %s", err)
	}
	t, err := template.New("").Parse(string(s))
	if err != nil {
		return nil, fmt.Errorf("Unable to generate template from marshalled YAML: %s", err)
	}
	return &Template{t}, nil
}

// InjectTemplateValues is a convenience func that takes an initial value, a destination
// value pointer, and a pipeline value, and converts the initialValue to a YAML template,
// then renders it using the provided pipeline value, and finally unmarshals it to the
// writeTo pointer. A typical invocation may look like this:
//
// err := InjectTemplatePipeline(v, &v, map[string]string{"hello":"world"})
//
// Note that in the above invocation the source value v is mutated, since it is passed
// in as the initial value, and then a pointer to it is passed in as the second value.
// This is a common pattern, but is optional, you can use two unrelated values if you want
// to preserve the initial value of v.
func InjectTemplatePipeline(initialValue, writeTo, values interface{}) error {
	template, err := NewTemplateFromValue(initialValue)
	if err != nil {
		return err
	}
	return template.SetPipeline(values).Render().Unmarshal(writeTo)
}

func (t *Template) SetPipeline(a interface{}) *TemplateWithPipeline {
	return &TemplateWithPipeline{t.Template, a}
}

func (t *TemplateWithPipeline) Render() *RenderedYAML {
	buf := &bytes.Buffer{}
	err := t.Template.Execute(buf, t.Pipeline)
	return &RenderedYAML{buf.Bytes(), err}
}

func (t *RenderedYAML) Unmarshal(v interface{}) error {
	if err := Unmarshal(t.Bytes, v); err != nil {
		return fmt.Errorf("Unable to unmarshal rendered YAML template: %s", err)
	}
	return nil
}
