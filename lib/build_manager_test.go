package sous

import "testing"

func rootedBuildManager(root string) *BuildManager {
	return &BuildManager{
		BuildConfig: &BuildConfig{
			Context: &BuildContext{
				Source: SourceContext{
					RootDir: root,
				},
			},
		},
	}
}

func TestOffsetFromWorkdir(t *testing.T) {
	root := "/somewhere/project"
	relWork := "/working"
	cliArg := "offset"
	off := "working/offset"
	bm := rootedBuildManager(root)

	err := bm.OffsetFromWorkdir(root+"/"+relWork, cliArg)
	if err != nil {
		t.Errorf("error not nil: %v", err)
	}
	if bm.BuildConfig.Offset != off {
		t.Errorf("%q != %q", bm.BuildConfig.Offset, off)
	}
}

func TestOffsetFromWorkdirOffsetOutsideOfRoot(t *testing.T) {
	root := "/somewhere/project"
	cliArg := ".."
	bm := rootedBuildManager(root)

	err := bm.OffsetFromWorkdir(root, cliArg)
	if err == nil {
		t.Errorf("error nil when offset outside of project root")
	}
	if bm.BuildConfig.Offset != "" {
		t.Errorf("Offset = %q", bm.BuildConfig.Offset)
	}
}

func TestOffsetFromWorkdirOffsetWhenWorkDirIsBad(t *testing.T) {
	root := "/somewhere/project"
	workdir := "you/know/around"
	cliArg := ".."
	bm := rootedBuildManager(root)

	err := bm.OffsetFromWorkdir(workdir, cliArg)
	if err == nil {
		t.Errorf("error nil when workdir isn't absolute")
	}
	if bm.BuildConfig.Offset != "" {
		t.Errorf("Offset = %q", bm.BuildConfig.Offset)
	}
}

func TestOffsetFromWorkdirWhenOffsetAlreadyConfigd(t *testing.T) {
	root := "/somewhere/project"
	relWork := "/working"
	cliArg := "offset"
	off := "working/offset"
	flagOffset := "working/elsewhere"
	bm := rootedBuildManager(root)
	bm.BuildConfig.Offset = flagOffset
	err := bm.OffsetFromWorkdir(root+"/"+relWork, cliArg)

	if err == nil {
		t.Errorf("error nil when offset already set")
	}
	if bm.BuildConfig.Offset != flagOffset {
		t.Errorf("%q = %q (not %q)", bm.BuildConfig.Offset, off, flagOffset)
	}
}

type FakeRegistrar struct{}

func (FakeRegistrar) Register(*BuildResult, *BuildContext) error { return nil }

func TestBuildManager_RegisterAndWarnAdvisories_withAdvisories(t *testing.T) {
	br := &BuildResult{}
	bc := &BuildContext{
		Advisories: []string{"dirty workspace"},
	}
	m := &BuildManager{
		Registrar: FakeRegistrar{},
	}
	if err := m.RegisterAndWarnAdvisories(br, bc); err != nil {
		t.Fatal(err)
	}
}

func TestBuildManager_RegisterAndWarnAdvisories_noAdvisories(t *testing.T) {
	br := &BuildResult{}
	bc := &BuildContext{
		Advisories: []string{},
	}
	m := &BuildManager{
		Registrar: FakeRegistrar{},
	}
	if err := m.RegisterAndWarnAdvisories(br, bc); err != nil {
		t.Fatal(err)
	}
}
