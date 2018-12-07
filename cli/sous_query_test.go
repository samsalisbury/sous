package cli

import "testing"

func TestSousQuery(t *testing.T) {
	c := &SousQuery{}
	gotExitCode := c.Execute(nil).ExitCode()
	const wantExitCode = 64
	if gotExitCode != wantExitCode {
		t.Errorf("got exit code %d; want %d", gotExitCode, wantExitCode)
	}
}
