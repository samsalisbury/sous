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
	"github.com/pkg/errors"
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
	log.Printf("Templating %q -> %q.", sourceDir, targetDir)
	var linkCount, templCount int
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
			return errors.Wrap(err, "open")
		}

		if 0 != (info.Mode() & os.ModeSymlink) {
			linkT, err := os.Readlink(path)
			if err != nil {
				return errors.Wrap(err, "readlink")
			}
			if filepath.IsAbs(linkT) {
				linkT, err = filepath.Rel(sourceDir, linkT)
				if err != nil {
					return errors.Wrap(err, "Rel link")
				}
			}
			linkName := filepath.Join(targetDir, info.Name())
			linkCount++
			return errors.Wrap(os.Symlink(linkT, linkName), "create link")
		}

		sourcePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return errors.Wrap(err, "Rel file")
		}

		bytes, err := ioutil.ReadAll(f)
		if err != nil {
			return errors.Wrap(err, "read")
		}

		targetPath := filepath.Join(targetDir, sourcePath)
		err = os.MkdirAll(filepath.Dir(targetPath), os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "create dir")
		}

		target, err := os.Create(targetPath)
		if err != nil {
			return errors.Wrap(err, "create target")
		}
		defer target.Close()

		tmpl, err := template.New(f.Name()).Parse(string(bytes))
		if err != nil {
			return errors.Wrap(err, "parse")
		}

		templCount++
		return errors.Wrap(tmpl.Execute(target, configData), "execute")
	})
	log.Printf("Linked %d files, Templated %d files.", linkCount, templCount)
	return err
}

func withHostEnv(hostEnvs []string, env map[string]string) map[string]string {
	newEnv := make(map[string]string)
	for _, k := range hostEnvs {
		newEnv[k] = os.Getenv(k)
	}
	for k, v := range env {
		newEnv[k] = v
	}
	return newEnv
}

type templatedConfigs struct {
	desc.EnvDesc
	TestDir, Workdir, Homedir, Statedir string
	XDGConfig, SSHWrapper               string
	GitSSH, GitRemoteBase               string
	SSHExec                             string
	GoPath, ShellPath                   []string
}

func setupConfig(t *testing.T) templatedConfigs {
	descPath := os.Getenv("SOUS_QA_DESC")
	if descPath == "" {
		t.Fatalf("SOUS_QA_DESC is empty - you need to run sous_qa_setup and set that env var")
	}

	pwd := filepath.Dir(descPath)

	envDesc, err := desc.LoadDesc(descPath)
	if err != nil {
		t.Fatalf("Couldn't load a QA env description from SOUS_QA_DESC(%q): %s", descPath, err)
	}
	if !envDesc.Complete() {
		t.Fatal("Incomplete QA env description. Re-run sous_qa_setup?")
	}

	workdir, err := ioutil.TempDir("", "sous-cli-testing")
	if err != nil {
		t.Fatalf("Couldn't create temporary working directory: %s", err)
	}

	sousExeDir := filepath.Join(workdir, "sous", "bin")
	sousExe := filepath.Join(sousExeDir, "sous")
	if out, err := exec.Command("go", "build", "-o", sousExe, "..").CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}

	//defer os.RemoveAll(workdir)

	stateDir := filepath.Join(workdir, "gdm")

	exePATH, err := buildPath("go", "git", "ssh", "cp")
	if err != nil {
		t.Fatal(err)
	}

	sshExecPath, err := exec.LookPath("ssh")
	if err != nil {
		t.Fatal(err)
	}

	testHome := filepath.Join(workdir, "home")

	gitRemoteBase := `ssh://root@` + envDesc.GitOrigin + "/repos"
	gitSSH := envDesc.AgentIP.String()

	sshWrapper := filepath.Join(testHome, "bin/ssh_wrapper")
	firstGoPath := filepath.Join(testHome, "go")

	shellPath := []string{sousExeDir, "~/bin", exePATH, filepath.Join(firstGoPath, "bin")}

	goPath := []string{firstGoPath}

	if userGopath := os.Getenv("GOPATH"); userGopath != "" {
		for _, userGo := range strings.Split(userGopath, ":") {
			goPath = append(goPath, userGo)
			shellPath = append(shellPath, filepath.Join(userGo, "bin"))
		}
	}

	return templatedConfigs{
		TestDir:       pwd,
		EnvDesc:       envDesc,
		Workdir:       workdir,
		Homedir:       testHome,
		Statedir:      stateDir,
		XDGConfig:     filepath.Join(testHome, "dot-config"),
		SSHWrapper:    sshWrapper,
		GoPath:        goPath,
		GitSSH:        gitSSH,
		SSHExec:       sshExecPath,
		GitRemoteBase: gitRemoteBase,
		ShellPath:     shellPath,
	}
}

func buildShell(name string, t *testing.T) *shelltest.ShellTest {
	cfg := setupConfig(t)

	os.MkdirAll(cfg.Homedir, os.ModePerm)
	err := templateConfigs(filepath.Join(cfg.TestDir, "integration/test-homedir"), cfg.Homedir, cfg)
	if err != nil {
		t.Fatalf("Templating configuration files: %+v", err)
	}

	shell := shelltest.New(t, name, cfg,
		withHostEnv([]string{"DOCKER_HOST", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH"},
			map[string]string{
				"HOME":       cfg.Homedir,
				"XDG_CONFIG": cfg.XDGConfig,
				"GIT_SSH":    cfg.SSHWrapper,
				"GOPATH":     strings.Join(cfg.GoPath, ":"),
				"PATH":       strings.Join(cfg.ShellPath, ":"),
			}))

	shell.WriteTo("../doc/shellexamples")
	shell.DebugPrefix("shell")

	return shell
}

func TestShellLevelIntegration(t *testing.T) {
	shell := buildShell("happypath", t)

	defaultCheck := func(name string, res shelltest.Result, t *testing.T) {
		if len(res.Errs) > 0 {
			t.Errorf("Error in %s: \n\t%s", name, res.Errs)
		}
	}

	preconditions := shell.Block("Preconditions for CLI integration tests", `
	go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
	cygnus -H {{.EnvDesc.SingularityURL}}
	ls {{.TestDir}}/dev_support
	`, func(name string, res shelltest.Result, t *testing.T) {
		if len(res.Errs) > 0 {
			t.Errorf("Error in %s: \n\t%s", name, res.Errs)
		}
		if res.Matches("sous-server") {
			msg, err := shell.Template("clean-sing", "Running sous-server already - try `./integration/test-registry/clean-singularity.sh {{.EnvDesc.SingularityURL}}`")
			if err != nil {
				t.Error("Running sous-server already - try `./integration/test-registry/clean-singularity.sh <singularity-url>`")
			}
			t.Error(msg)
		}
		if !res.Matches("sous_linux") {
			t.Error("No sous_linux available - run `make linux_build`")
		}
	})

	prologue := preconditions.Block("Test environment setup", `
	# These steps are required by the Sous integration tests
	# They're analogous to run-of-the-mill workstation maintenance.

	cd {{.TestDir}}
	env
	export SOUS_EXTRA_DOCKER_CA={{.TestDir}}/integration/test-registry/docker-registry/testing.crt
	mkdir -p {{index .GoPath 0}}/{src,bin}

	### This build gives me trouble in tests...
	### xgo does something weird and different with it's dep-cache dir
	# GOPATH={{index .GoPath 0}} make linux_build # we need Sous built for linux for the server
	go install . #install the current sous project
	cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/

	cd {{.Workdir}}
	chmod go-rwx -R ~/dot-ssh
	chmod +x -R ~/bin/*
	ssh -o ConnectTimeout=1 -o PasswordAuthentication=no -F "${HOME}/dot-ssh/config" root@{{.GitSSH}} -p 2222 /reset-repos < /dev/null
	`,
		defaultCheck)

	createGDM := prologue.Block("create the GDM", `
	git clone {{.GitRemoteBase}}/gdm
	cp ~/templated-configs/defs.yaml gdm/defs.yaml
	cat gdm/defs.yaml
	pushd gdm
	cat ~/.config/git/config >> .git/config # Eh?
	git add defs.yaml
	git commit -am "Adding defs.yaml"
	git push
	popd
	`, defaultCheck)

	// XXX There should be a `-cluster left,right` syntax, instead of two deploy commands
	setup := createGDM.Block("deploy sous server", `
	sous config
	cat ~/.config/sous/config.yaml
	git clone {{.GitRemoteBase}}/sous-server
	pushd sous-server
	export SOUS_USER_NAME=test SOUS_USER_EMAIL=test@test.com
	export SOUS_SERVER= SOUS_STATE_LOCATION={{.Statedir}}
	sous init -v -d
	sous manifest get | sed '/version/a\
	\    env:
	/version/a\
	\      GDM_REPO: "{{.GitRemoteBase}}/gdm"
	' > ~/sous-server.yaml
	cat ~/sous-server.yaml
	sous manifest set < ~/sous-server.yaml

	# Last minute config
	cat Dockerfile
	cp ~/dot-ssh/git_pubkey_rsa key_sous@example.com
	cp {{.TestDir}}/dev_support/$(readlink {{.TestDir}}/dev_support/sous_linux) .
	ls -la
	ssh-keyscan -p 2222 {{.GitSSH}} > known_hosts
	cat known_hosts

	git add key_sous@example.com known_hosts sous
	git commit -am "Adding ephemeral files"
	git tag -am "0.0.2" 0.0.2
	git push
	git push --tags
	sous context
	pwd
	sous build
	# We expect to see 'Sous is running ... in workstation mode' here:
	sous deploy -cluster left
	sous deploy -cluster right
	unset SOUS_SERVER
	unset SOUS_STATE_LOCATION
	popd
	`,
		func(name string, res shelltest.Result, t *testing.T) {
			if len(res.Errs) > 0 {
				t.Errorf("Trouble building GDM: \n\t%s", res.Errs)
			}

			if !res.Matches(`--author test <test@`) {
				t.Errorf("SOUS_USER_NAME not used for local state updates")
			}
		})

	config := setup.Block("configuration", `
	cygnus --env TASK_HOST --env PORT0 {{.EnvDesc.SingularityURL}}
	serverURL=$(cygnus --env TASK_HOST --env PORT0 {{.EnvDesc.SingularityURL}} | grep 'sous-server.*left' | awk '{ print "http://" $3 ":" $4 }')
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
	git clone {{.GitRemoteBase}}/sous-demo
	cd sous-demo
	git tag -a 0.0.23
	git push --tags
	sous init
	sous build
	sous deploy -cluster left
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
