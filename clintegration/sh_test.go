package clintegration

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/util/shelltest"
)

// XXX move to shelltest
func TestShAssumptions(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	shell, err := shelltest.NewShell(nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := shell.Run(`
	cd /tmp
	X=7
	export CYGNUS=blackhole
	echo $X
	pwd
	`)

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !res.Matches(`7`) {
		t.Errorf("No 7")
	}
	if !res.Matches(`/tmp`) {
		t.Errorf("Not in /tmp")
	}

	res, err = shell.Run(`
	echo $X
	pwd
	env
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !res.Matches(`7`) {
		t.Errorf("No 7")
	}
	if !res.Matches(`/tmp`) {
		t.Errorf("Not in /tmp")
	}
}

func buildPath(exes ...string) (string, error) {
	dirMap := map[string]struct{}{}

	for _, name := range exes {
		exePath, err := exec.LookPath(name)
		if err != nil {
			return "", err
		}

		dirMap[filepath.Dir(exePath)] = struct{}{}
	}

	dirs := []string{}
	for path := range dirMap {
		dirs = append(dirs, path)
	}

	return strings.Join(dirs, ":"), nil
}

func templateConfigs(sourceDir, targetDir string, configData templatedConfigs) error {
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}

		sourcePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		bytes, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, sourcePath)

		target, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer target.Close()

		tmpl, err := template.New(f.Name()).Parse(string(bytes))
		if err != nil {
			return err
		}

		return tmpl.Execute(target, configData)
	})
	return err
}

type templatedConfigs struct {
	desc.EnvDesc
	Workdir string
}

// XXX Do we need a separate test for the test infra?
func TestTemplating(t *testing.T) {
	descPath := os.Getenv("SOUS_QA_DESC")
	if descPath == "" {
		t.Fatalf("SOUS_QA_DESC is empty - you need to run sous_qa_setup and set that env var")
	}

	pwd := filepath.Dir(descPath)

	envDesc, err := desc.LoadDesc(descPath)
	if err != nil {
		t.Fatalf("Couldn't load a QA env description from SOUS_QA_DESC(%q): %s", descPath, err)
	}

	tmpDir, err := ioutil.TempDir("", "sous-cli-templating")
	if err != nil {
		t.Fatalf("Couldn't create temporary working directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := templatedConfigs{
		EnvDesc: envDesc,
		Workdir: "/working/dir",
	}

	err = templateConfigs(filepath.Join(pwd, "integration/test-config-templates"), tmpDir, cfg)
	if err != nil {
		t.Error(err)
	}
}

func TestShellLevelIntegration(t *testing.T) {
	descPath := os.Getenv("SOUS_QA_DESC")
	if descPath == "" {
		t.Fatalf("SOUS_QA_DESC is empty - you need to run sous_qa_setup and set that env var")
	}

	envDesc, err := desc.LoadDesc(descPath)
	if err != nil {
		t.Fatalf("Couldn't load a QA env description from SOUS_QA_DESC(%q): %s", descPath, err)
	}

	workdir, err := ioutil.TempDir("", "sous-cli-testing")
	if err != nil {
		t.Fatalf("Couldn't create temporary working directory: %s", err)
	}
	log.Printf("WORKDIR: %q", workdir)
	//defer os.RemoveAll(workdir)

	stateDir := filepath.Join(workdir, "gdm")

	pwd := filepath.Dir(descPath)

	exePATH, err := buildPath("go", "git", "ssh", "cp")

	testHome := filepath.Join(workdir, "home")

	gitRemoteBase := `ssh://root@` + envDesc.GitOrigin + "/repos"
	gitSSH := strings.Split(envDesc.GitOrigin, ":")[0]

	cfg := templatedConfigs{
		EnvDesc: envDesc,
		Workdir: workdir,
	}

	tmplTgt := filepath.Join(workdir, "templated-configs")
	os.MkdirAll(tmplTgt, os.ModePerm)
	err = templateConfigs(filepath.Join(pwd, "integration/test-config-templates"), tmplTgt, cfg)
	if err != nil {
		t.Fatalf("Templating configuration files: %s", err)
	}

	firstGoPath := filepath.Join(testHome, "go")
	goPath := firstGoPath
	if userGopath := os.Getenv("GOPATH"); userGopath != "" {
		goPath = goPath + ":" + userGopath
	}

	shell := shelltest.New(t, "happypath", map[string]string{
		"HOME":    testHome,
		"GIT_SSH": filepath.Join(testHome, "bin/ssh_wrapper"),
		"GOPATH":  goPath,
		"PATH":    strings.Join([]string{"~/bin", exePATH, filepath.Join(firstGoPath, "bin")}, ":"),
	})

	shell.WriteTo("../doc/shellexamples")

	defaultCheck := func(name string, res shelltest.Result, t *testing.T) {
		if len(res.Errs) > 0 {
			t.Errorf("Error in %s: \n\t%s", name, res.Errs)
		}
	}

	prologue := shell.Block("Test environment setup", `
	# These steps are required by the Sous integration tests
	# They're analogous to run-of-the-mill workstation maintenance.

	env
	mkdir -p `+firstGoPath+`/{src,bin}
	go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
	cd `+pwd+`
	go install . #install the current sous project
	cp -a integration/test-homedir/* "$HOME"
	cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/
	cd `+workdir+`
	cp templated-configs/ssh-config ~/dot-ssh/config
	chmod go-rwx -R ~/dot-ssh
	git config --global --add user.name "Integration Tester"
	git config --global --add user.email "itester@example.com"
	echo ~/bin/ssh_wrapper root@`+gitSSH+` -p 2222 /reset-repos
	`,
		defaultCheck)

	createGDM := prologue.Block("create the GDM", `
	ls -l
	git clone `+gitRemoteBase+`/gdm
	ls -l
	cat templated-configs/defs.yaml | tee gdm/defs.yaml
	pushd gdm
	git add defs.yaml
	git commit -am "Adding defs.yaml"
	git push
	popd
	`, defaultCheck)

	// XXX There should be a `-cluster left,right` syntax, instead of two deploy commands
	setup := createGDM.Block("sous setup", `
	git clone `+gitRemoteBase+`/sous-server
	ls -l
	pushd sous-server
	sous init
	sous build
	# We expect to see 'Sous is running ... in workstation mode' here:
	SOUS_SERVER= SOUS_STATE_LOCATION=`+stateDir+` sous deploy -cluster left
	SOUS_SERVER= SOUS_STATE_LOCATION=`+stateDir+` sous deploy -cluster right
	popd
	`,
		func(name string, res shelltest.Result, t *testing.T) {
			if len(res.Errs) > 0 {
				t.Errorf("Trouble building GDM: \n\t%s", res.Errs)
			}

			if !res.Matches(`Deployed`) {
				t.Errorf("No report of deployment")
			}
		})

	// XXX Event driven wait for the server to be ready?

	config := setup.Block("configuration", `
	serverURL=$(cygnus --env TASK_HOST --env PORT0 `+envDesc.SingularityURL+` | grep 'sous-server.*left' | awk '{ print "http://" $3 ":" $4 }')
	sous config Server "$serverURL"
	echo -n "Server URL is: "
	sous config Server
	`,
		func(name string, res shelltest.Result, t *testing.T) {
			if len(res.Errs) > 0 {
				t.Errorf("Trouble building GDM: \n\t%s", res.Errs)
			}

			if !res.Matches(`URL is: http`) {
				t.Fatalf("Sous server not running!")
			}
		})

	deploy := config.Block("deploy project", `
	git clone `+gitRemoteBase+`/sous-demo
	cd sous-demo
	git tag -a 0.0.23
	git push --tags
	sous init
	sous build
	sous deploy -tag 0.0.23
	`, defaultCheck)

	//check :=
	deploy.Block("confirm deployment", `
	cygnus -x 1 | grep sous-demo
	`, func(name string, res shelltest.Result, t *testing.T) {
		if res.Exit != 0 {
			t.Errorf("No match for 'sous-demo' in names of running requests")
		}
	})
}
