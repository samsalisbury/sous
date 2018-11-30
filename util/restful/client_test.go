package restful

import (
	"fmt"
	"testing"
)

func TestDummyHTTPClient(t *testing.T) {

	assertSameErrors := func(got, want error) {
		t.Helper()
		if got == nil && want == nil {
			return // Success.
		}
		if want == nil {
			t.Fatalf("got error %q; want nil", got)
		}
		if got == nil {
			t.Fatalf("got nil; want error %q", want)
		}
		g := got.Error()
		w := want.Error()
		if g != w {
			t.Errorf("got error %q; want %q", g, w)
		}
	}

	cases := []error{nil, fmt.Errorf("error1"), fmt.Errorf("error2")}

	for _, c := range cases {
		t.Run(fmt.Sprintf("AlwaysReturnErr=%v", c), func(t *testing.T) {
			client := DummyHTTPClient{AlwaysReturnErr: c}
			t.Run("Create", func(t *testing.T) {
				_, gotErr := client.Create("", nil, nil, nil)
				assertSameErrors(c, gotErr)
			})
			t.Run("Retrieve", func(t *testing.T) {
				_, gotErr := client.Retrieve("", nil, nil, nil)
				assertSameErrors(c, gotErr)
			})
			t.Run("update", func(t *testing.T) {
				gotErr := client.update("", nil, nil, nil, nil)
				assertSameErrors(c, gotErr)
			})
			t.Run("deelete", func(t *testing.T) {
				gotErr := client.deelete("", nil, nil, nil)
				assertSameErrors(c, gotErr)
			})
		})
	}
}
