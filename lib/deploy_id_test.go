package sous

import "testing"

var deployIDTests = []struct {
	String   string
	DeployID DeployID
}{
	{
		String: "github.com/user/repo:some-cluster",
		DeployID: DeployID{
			Cluster: "some-cluster",
			ManifestID: ManifestID{
				Source: SourceLocation{
					Repo: "github.com/user/repo",
				},
			},
		},
	},
	{
		String: "github.com/user/repo,some-dir:some-cluster",
		DeployID: DeployID{
			Cluster: "some-cluster",
			ManifestID: ManifestID{
				Source: SourceLocation{
					Repo: "github.com/user/repo",
					Dir:  "some-dir",
				},
			},
		},
	},
	{
		String: "github.com/user/repo,some-dir~british-flavoured:some-cluster",
		DeployID: DeployID{
			Cluster: "some-cluster",
			ManifestID: ManifestID{
				Source: SourceLocation{
					Repo: "github.com/user/repo",
					Dir:  "some-dir",
				},
				Flavor: "british-flavoured",
			},
		},
	},
}

func TestParseDeployID(t *testing.T) {
	for _, test := range deployIDTests {
		input := test.String
		expected := test.DeployID
		actual, err := ParseDeployID(input)
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%q got %#v; want %#v", input, actual, expected)
		}
	}
}

//func TestDeployID_String(t *testing.T) {
//	for _, test := range deployIDTests {
//		expected := test.String
//		actual := test.DeployID.String()
//		if actual != expected {
//			t.Errorf("%#v got %q; want %q", test.DeployID, actual, expected)
//		}
//	}
//}
//
//func TestDeployID_MarshalText(t *testing.T) {
//	for _, test := range deployIDTests {
//		input := test.DeployID
//		expected := test.String
//		actualBytes, err := input.MarshalText()
//		if err != nil {
//			t.Error(err)
//			continue
//		}
//		actual := string(actualBytes)
//		if actual != expected {
//			t.Errorf("%#v got %q; want %q", input, actual, expected)
//		}
//	}
//}
//
//func TestDeployID_MarshalYAML(t *testing.T) {
//	for _, test := range deployIDTests {
//		input := test.DeployID
//		expected := test.String
//		actualInterface, err := input.MarshalYAML()
//		if err != nil {
//			t.Error(err)
//			continue
//		}
//		actual := actualInterface.(string)
//		if actual != expected {
//			t.Errorf("%#v got %q; want %q", input, actual, expected)
//		}
//	}
//}
//
//func TestDeployID_UnmarshalText(t *testing.T) {
//	for _, test := range deployIDTests {
//		input := []byte(test.String)
//		expected := test.DeployID
//		var actual DeployID
//		if err := actual.UnmarshalText(input); err != nil {
//			t.Error(err)
//			continue
//		}
//		if actual != expected {
//			t.Errorf("%q got %#v; want %#v", input, actual, expected)
//		}
//	}
//}
//
//func TestDeployID_UnmarshalYAML(t *testing.T) {
//	for _, test := range deployIDTests {
//		inputStr := test.String
//		expected := test.DeployID
//		var actual DeployID
//		// inputFunc mocks out behaviour of the github.com/go-yaml/yaml library.
//		inputFunc := func(v interface{}) error {
//			sp := reflect.ValueOf(v)
//			if sp.Kind() != reflect.Ptr {
//				return fmt.Errorf("got %s; want *string", sp.Type())
//			}
//			s := sp.Elem()
//			if s.Kind() != reflect.String {
//				return fmt.Errorf("got %s; want a *string", sp.Type())
//			}
//			s.Set(reflect.ValueOf(inputStr))
//			return nil
//		}
//		if err := actual.UnmarshalYAML(inputFunc); err != nil {
//			t.Error(err)
//			continue
//		}
//		if actual != expected {
//			t.Errorf("%q got %#v; want %#v", inputStr, actual, expected)
//		}
//	}
//}
