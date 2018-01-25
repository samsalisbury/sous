package server

import (
	"testing"

	"github.com/samsalisbury/semv"
)

func TestHandleHealth_Get(t *testing.T) {
	version := "3.4.5"

	h := &getHealthHandler{
		version: semv.MustParse(version),
	}
	data, stat := h.Exchange()

	if stat != 200 {
		t.Errorf("Expecting 200 status; got %d", stat)
	}

	rez, is := data.(Health)

	if !is {
		t.Fatalf("getHealthHandler didn't return a Health struct, but instead a %T: %[1]q", data)
	}

	if rez.Version != version {
		t.Errorf("Expecting %q; got %q", version, rez.Version)
	}
}
