package test_with_docker

import (
	"fmt"
	"log"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Machine struct {
	name           string
	serviceTimeout float32
}

func (m *Machine) ComposeServices(dir string, servicePorts serviceMap) (shutdown *command, err error) {
	ip, err := m.IP()
	if err != nil {
		return nil, err
	}
	env := m.env()

	return composeService(dir, ip, env, servicePorts, m.serviceTimeout)
}

func (m *Machine) DifferingFiles(pathPairs ...[]string) (differentPairs [][]string, err error) {
	localPaths, remotePaths := make([]string, 0, len(pathPairs)), make([]string, 0, len(pathPairs))

	for _, pair := range pathPairs {
		localPaths = append(localPaths, pair[0])
		remotePaths = append(remotePaths, pair[1])
	}

	localMD5s := localMD5s(localPaths...)
	remoteMD5s, err := m.MD5s(remotePaths...)
	if err != nil {
		return
	}

	return fileDiffs(pathPairs, localMD5s, remoteMD5s), nil
}

func (m *Machine) IP() (ip net.IP, err error) {
	alreadyStarted := regexp.MustCompile("is already running")
	_, stderr, err := dockerMachine("start", m.name)
	if err != nil && !alreadyStarted.MatchString(stderr) {
		log.Panic("start", err)
	}

	ipStr, _, err := dockerMachine("ip", m.name)
	if err != nil {
		return
	}
	ipStr = strings.Trim(ipStr, " \n\t")

	ip = net.ParseIP(ipStr)
	return
}

func (m *Machine) RebuildService(dir, name string) error {
	env := m.env()
	return rebuildService(dir, name, env)
}

func (m *Machine) MD5s(paths ...string) (md5s map[string]string, err error) {
	stdout, stderr, err := dockerMachine(append([]string{"ssh", m.name, "sudo", "md5sum"}, paths...)...)
	md5s = make(map[string]string)

	if err != nil {
		allMatches := md5missingRE.FindAllStringSubmatch(stderr, -1)
		for _, matches := range allMatches {
			if len(matches[1]) > 0 {
				md5s[matches[1]] = ""
			} else {
				md5s[matches[2]] = ""
			}
		}
		newPaths := paths[:0]
		for _, path := range paths {
			if _, missing := md5s[path]; !missing {
				newPaths = append(newPaths, path)
			}
		}

		err = nil
		if len(newPaths) > 0 {
			stdout, stderr, err = dockerMachine(append([]string{"ssh", m.name, "sudo", "md5sum"}, newPaths...)...)
			if err != nil {
				md5s = nil
				return
			}
		}
	}

	allMatches := md5RE.FindAllStringSubmatch(stdout, -1)
	for _, matches := range allMatches {
		md5s[matches[2]] = matches[1]
	}

	return
}

func tempFilePath() string {
	//stolen from ioutil.tempfile
	return "/tmp/temp-" + strconv.Itoa(int(1e9 + rnums.Int31()%1e9))[1:]
}

func (m *Machine) InstallFile(sourcePath, destPath string) error {
	tmpFile := tempFilePath()
	destDir := filepath.Dir(destPath)
	scpTmp := fmt.Sprintf("%s:%s", m.name, tmpFile)
	if _, _, err := dockerMachine("scp", sourcePath, scpTmp); err != nil {
		return err
	}
	if err := m.Exec("mkdir", "-p", destDir); err != nil {
		return err
	}
	return m.Exec("mv", tmpFile, destPath)
}

func (m *Machine) RestartDaemon() error {
	return m.Exec("/etc/init.d/docker", "restart")
}

// DockerMachineSshSudo is your out for anything that test_with_docker doesn't provide.
// It executes `docker-machine ssh <machineName> sudo <args...>` so that you
// can manipulate the running docker machine
func (m *Machine) Exec(args ...string) error {
	_, _, err := dockerMachine(append([]string{"ssh", m.name, "sudo"}, args...)...)
	return err
}

// Shutdown receives a command as produced by ComposeServices is shuts down
// services launched for testing.
// If passed a nil command, it functions as a no-op. This means that you can
// do things like:
//   ip, cmd, err := ComposeServices(...)
//   defer Shutdown(cmd)
func (m *Machine) Shutdown(c *command) {
	if c != nil {
		dockerComposeDown(c)
	}
}

func dockerMachine(args ...string) (stdoutStr, stderrStr string, err error) {
	dmCmd := runCommand("docker-machine", args...)
	log.Print(dmCmd.String(), "\n")
	return dmCmd.stdout, dmCmd.stderr, dmCmd.err
}

func (m *Machine) env() []string {
	setPrefix := regexp.MustCompile("^SET ")
	envStr, _, err := dockerMachine("env", "--shell", "cmd", m.name) //other shells get extraneous quotes
	if err != nil {
		log.Panic("env", err)
	}

	env := make([]string, 0, 4)
	for _, str := range strings.Split(envStr, "\n") {
		env = append(env, setPrefix.ReplaceAllString(str, ""))
	}

	return env
}
