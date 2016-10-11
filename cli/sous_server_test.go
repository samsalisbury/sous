package cli

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
)

// TODO: This would be better as separate tests for each case.
func TestEnsureGDMExists(t *testing.T) {
	testDataDir := "testdata/gen/test-ensure-gdm-exists"

	if err := os.RemoveAll(testDataDir); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	gdmSourceRepo := path.Join(wd, testDataDir, "test-repo")
	nonexistentSourceLocation := path.Join(testDataDir, "does-not-exist")
	emptySourceLocation := path.Join(testDataDir, "empty-source-location")
	nonemptySourceLocation := path.Join(testDataDir, "nonempty-source-location")
	someFile := path.Join(testDataDir, "a-file")

	if err := os.MkdirAll(gdmSourceRepo, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(emptySourceLocation, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nonemptySourceLocation, 0777); err != nil {
		t.Fatal(err)
	}
	nonemptyFile := path.Join(nonemptySourceLocation, "another-file")
	if err := ioutil.WriteFile(nonemptyFile, []byte("hi"), 0777); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(someFile, []byte("hi"), 0777); err != nil {
		t.Fatal(err)
	}

	// Create the empty source repo.
	c := exec.Command("git", "init")
	c.Dir = gdmSourceRepo
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path.Join(gdmSourceRepo, "a-file"), []byte("hi"), 0777); err != nil {
		t.Fatal(err)
	}
	c = exec.Command("git", "add", "-A")
	c.Dir = gdmSourceRepo
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	c = exec.Command("git", "commit", "-m", "a message")
	c.Dir = gdmSourceRepo
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}

	// SourceLocation dir does not exist; repo does.
	if err := ensureGDMExists(gdmSourceRepo, nonexistentSourceLocation, t.Logf); err != nil {
		t.Error(err)
	}

	// SourceLocation dir exists and is empty.
	if err := ensureGDMExists(gdmSourceRepo, emptySourceLocation, t.Logf); err != nil {
		t.Error(err)
	}

	// SourceLocation dir exists and is not empty.
	if err := ensureGDMExists(gdmSourceRepo, nonemptySourceLocation, t.Logf); err != nil {
		t.Error(err)
	}

	// Cleanup
	if err := os.RemoveAll(nonexistentSourceLocation); err != nil {
		t.Fatal(err)
	}
	// GDM source repo does not exist.
	{
		err := ensureGDMExists("this-path-does-not-exist", "", t.Logf)
		if err == nil {
			t.Error("expected error when GDM repo does not exist")
		}
	}

	// Cleanup.
	if err := os.RemoveAll(nonexistentSourceLocation); err != nil {
		t.Fatal(err)
	}
	// GDM source repo is not a repo.
	{
		err := ensureGDMExists(someFile, "", t.Logf)
		if err == nil {
			t.Error("expected error when GDM repo does not exist")
		}
	}

}
