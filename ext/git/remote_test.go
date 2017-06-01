package git

import "testing"

func TestCanonicalRepoURL_GoodInput(t *testing.T) {
	urls := []string{
		"https://github.com/user/project.git",
		"http://github.com/user/project.git",
		"https://github.com/user/project",
		"http://github.com/user/project",
		"git@github.com:user/project.git",
		"git@github.com:user/project",
		"ssh://git@github.com:user/project.git",
		"ssh://git@github.com:user/project",
		"ssh://github.com:user/project.git",
		"ssh://github.com:user/project",
		"git://git@github.com:user/project.git",
		"git://git@github.com:user/project",
		"git://github.com:user/project.git",
		"git://github.com:user/project",
		//"github.com/user/project",
	}
	expected := "github.com/user/project"
	for _, input := range urls {
		actual, err := CanonicalRepoURL(input)
		if err != nil {
			t.Errorf("Unexpected error %s", err)
			continue
		}
		if actual != expected {
			t.Errorf("Got %s as Canonical name for %s; want %s", actual, input, expected)
		}
	}
}

func TestCanonicalRepoURL_BadInput(t *testing.T) {
	urls := []string{
		//"https//github.com/user/project.git",
		"http:/github.com/user/project.git",
		"https:://github.com/user/project",
		"/github.com/user/project",
		//"git@github.comuser/project.git",
		"gitgithub.com:user/project",
		"::::::::::::",
	}
	for _, input := range urls {
		actual, err := CanonicalRepoURL(input)
		if err == nil {
			t.Errorf("%q should have caused an error, but canonicalised to %q", input, actual)
			continue
		}
		if actual != "" {
			t.Errorf("got %q for %q; want empty string", actual, input)
		}
		t.Log(err)
	}
}
