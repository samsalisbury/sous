package yaml

import "testing"

type Thing struct {
	StringField string
	StructField `yaml:",inline"`
}
type StructField struct {
	IntField int
}

func TestUnmarshalAnonymousFields(t *testing.T) {
	j := `{"StringField": "hello", "IntField": 5}`
	var it Thing
	if err := Unmarshal([]byte(j), &it); err != nil {
		t.Fatal(err)
	}
	if it.StructField.IntField != 5 {
		t.Fatalf("Nested field did not unmarshal.")
	}
	t.Logf("Got: % +v", it)
}
