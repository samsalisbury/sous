package yaml

import (
	"reflect"
	"testing"
)

func TestInjectTemplatePipelineMap(t *testing.T) {
	template := map[string]string{
		"k1":        "v1",
		"k2":        "{{.Value2}}",
		"{{.Key3}}": "{{.Value3}}",
	}
	pipeline := map[string]string{
		"Value2": "v2",
		"Value3": "v3",
		"Key3":   "k3",
	}
	expectedResult := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	var result map[string]string

	if err := InjectTemplatePipeline(template, &result, pipeline); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("Got result: % +v \n\nWant:\n\n% +v", result, expectedResult)
	}
}

type TestStruct struct {
	K1, K2 string
}

func TestInjectTemplatePipelineStruct(t *testing.T) {
	template := TestStruct{"v1", "{{.Value2}}"}
	pipeline := map[string]string{"Value2": "v2"}
	expectedResult := TestStruct{"v1", "v2"}
	var result TestStruct

	if err := InjectTemplatePipeline(template, &result, pipeline); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("Got result: % +v \n\nWant:\n\n% +v", result, expectedResult)
	}
}

type TestComplex struct {
	K1 map[string]map[string]string
	K2 TestStruct
}

func TestInjectTemplatePipelineComplex(t *testing.T) {
	template := TestComplex{
		K1: map[string]map[string]string{
			"k1k1": map[string]string{
				"k1k1k1":        "v1",
				"k1k1k2":        "{{.Value2}}",
				"k1k1{{.Key3}}": "v3",
				"k1k1{{.Key4}}": "{{.Value4}}",
			},
		},
		K2: TestStruct{
			K1: "structK1",
			K2: "{{.StructK2}}",
		},
	}
	pipeline := map[string]string{
		"Value2":   "v2",
		"Key3":     "k3",
		"Key4":     "k4",
		"Value4":   "v4",
		"StructK2": "structK2",
	}
	expectedResult := TestComplex{
		K1: map[string]map[string]string{
			"k1k1": map[string]string{
				"k1k1k1": "v1",
				"k1k1k2": "v2",
				"k1k1k3": "v3",
				"k1k1k4": "v4",
			},
		},
		K2: TestStruct{
			K1: "structK1",
			K2: "structK2",
		},
	}

	var result TestComplex

	if err := InjectTemplatePipeline(template, &result, pipeline); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("Got result: % +v \n\nWant:\n\n% +v", result, expectedResult)
	}
}

func TestInjectTemplatePipelineIntoSelfComplex(t *testing.T) {
	template := TestComplex{
		K1: map[string]map[string]string{
			"k1k1": map[string]string{
				"k1k1k1":        "v1",
				"k1k1k2":        "{{.Value2}}",
				"k1k1{{.Key3}}": "v3",
				"k1k1{{.Key4}}": "{{.Value4}}",
			},
		},
		K2: TestStruct{
			K1: "structK1",
			K2: "{{.StructK2}}",
		},
	}
	pipeline := map[string]string{
		"Value2":   "v2",
		"Key3":     "k3",
		"Key4":     "k4",
		"Value4":   "v4",
		"StructK2": "structK2",
	}
	expectedResult := TestComplex{
		K1: map[string]map[string]string{
			"k1k1": map[string]string{
				"k1k1k1": "v1",
				"k1k1k2": "v2",
				"k1k1k3": "v3",
				"k1k1k4": "v4",
			},
		},
		K2: TestStruct{
			K1: "structK1",
			K2: "structK2",
		},
	}

	if err := InjectTemplatePipeline(template, &template, pipeline); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(template, expectedResult) {
		t.Fatalf("Got result: % +v \n\nWant:\n\n% +v", template, expectedResult)
	}
}
