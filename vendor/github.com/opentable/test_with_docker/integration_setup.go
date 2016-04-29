// test_with_docker provides utilities for using docker-compose for writing
// integration tests.
package test_with_docker

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type command struct {
	itself         *exec.Cmd
	err            error
	stdout, stderr string
}

//	ComposeServices uses docker-compose to set up one or more services, using
//	serviceMap to check availability.
//
//	Importantly, the serviceMap is used both to determine if the services are
//	already available - since docker-compose can take some time to execute, it
//	can be handy to run the compose in a different console and let
//	ComposeServices discover the services.
//
//	Finally, if ComposeServices determined that a service was missing and
//	needed to be run, it will return a value that represents the
//	docker-compose command that it executed. You can pass this value to
//	Shutdown to shut down the docker-compose after tests have run.
func ComposeServices(machineName, dir string, servicePorts serviceMap) (ip string, shutdown *command, err error) {
	ip, err = MachineIP(machineName)

	if !servicesRunning(3.0, ip, servicePorts) {
		log.Printf("Services need to be started - tip: running `docker-compose up` in %s will speed tests up.", dir)

		shutdownCmd := dockerComposeUp(machineName, dir, ip, servicePorts)
		shutdown = &shutdownCmd
	} else {
		log.Printf("All services already up and running")
	}

	return
}

// MachineIP returns a string representing the IP address of a
// docker-machine.
// In order to access the services provided by a docker-compose on a
// docker-machine, we need to know the ip address. Some client test code
// needs to know the IP address prior to starting up the services, which is
// why this function is exposed
func MachineIP(machineName string) (ip string, err error) {
	alreadyStarted := regexp.MustCompile("is already running")
	_, stderr, err := dockerMachine("start", machineName)
	if err != nil && !alreadyStarted.MatchString(stderr) {
		log.Panic("start", err)
	}

	ip, _, err = dockerMachine("ip", machineName)
	if err != nil {
		return
	}
	ip = strings.Trim(ip, " \n\t")

	log.Print("Docker machine is running on ", ip)

	return
}

// RebuildService issues the docker-compose commands to ensure the
// rebuilding of a named service.
func RebuildService(machineName, dir, name string) error {
	cmd := buildCommand("docker-compose", "build", "--no-cache", name)
	cmd.itself.Env = machineEnv(machineName)
	cmd.itself.Dir = dir
	cmd.run()
	return cmd.err
}

var rnums = rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))

var md5RE = regexp.MustCompile(`(?m)^([0-9a-fA-F]+)\s+(\S+)$`)
var md5missingRE = regexp.MustCompile(`(?m)^md5sum: (?:can't open '(.*)'|(.*)): No such file or directory$`)

func MachineMD5s(machineName string, paths ...string) (md5s map[string]string, err error) {
	stdout, stderr, err := dockerMachine(append([]string{"ssh", machineName, "sudo", "md5sum"}, paths...)...)
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
			stdout, stderr, err = dockerMachine(append([]string{"ssh", machineName, "sudo", "md5sum"}, newPaths...)...)
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

// InstallMachineFile puts a path found on the host to a path on the docker-machine.
func InstallMachineFile(machineName, sourcePath, destPath string) error {
	tmpFile := "/tmp/temp-" + strconv.Itoa(int(1e9 + rnums.Int31()%1e9))[1:] //stolen from ioutil.tempfile
	destDir := filepath.Dir(destPath)

	scpTmp := fmt.Sprintf("%s:%s", machineName, tmpFile)
	_, _, err := dockerMachine("scp", sourcePath, scpTmp)
	if err != nil {
		return err
	}

	err = SshSudo(machineName, "mkdir", "-p", destDir)
	if err != nil {
		return err
	}

	err = SshSudo(machineName, "mv", tmpFile, destPath)
	if err != nil {
		return err
	}

	return nil
}

// DockerMachineSshSudo is your out for anything that test_with_docker doesn't provide.
// It executes `docker-machine ssh <machineName> sudo <args...>` so that you
// can manipulate the running docker machine
func SshSudo(machineName string, args ...string) error {
	_, _, err := dockerMachine(append([]string{"ssh", machineName, "sudo"}, args...)...)
	return err
}

// Shutdown receives a command as produced by ComposeServices is shuts down
// services launched for testing.
// If passed a nil command, it functions as a no-op. This means that you can
// do things like:
//   ip, cmd, err := ComposeServices(...)
//   defer Shutdown(cmd)
func Shutdown(c *command) {
	if c != nil {
		dockerComposeDown(c)
	}
}

func buildCommand(cmdName string, args ...string) command {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var c command
	c.itself = exec.Command(cmdName, args...)
	c.itself.Stdout = &stdout
	c.itself.Stderr = &stderr
	return c
}

func startCommand(cmdName string, args ...string) command {
	c := buildCommand(cmdName, args...)

	c.start()
	return c
}

func runCommand(cmdName string, args ...string) command {
	c := buildCommand(cmdName, args...)

	c.run()
	return c
}

func (c *command) start() error {
	c.err = c.itself.Start()
	return c.err
}

func (c *command) wait() error {
	c.err = c.itself.Wait()
	c.stdout = c.itself.Stdout.(*bytes.Buffer).String()
	c.stderr = c.itself.Stderr.(*bytes.Buffer).String()
	return c.err
}

func (c *command) run() error {
	c.start()
	if c.err != nil {
		return c.err
	}
	c.wait()

	return c.err
}

func (c *command) String() string {
	if c.err == nil {
		return fmt.Sprintf("%v ok", (*c.itself).Args)
	} else {
		return fmt.Sprintf("%v %v\nout: %serr: %s", (*c.itself).Args, c.err, c.stdout, c.stderr)
	}
}

func dockerMachine(args ...string) (stdoutStr, stderrStr string, err error) {
	dmCmd := runCommand("docker-machine", args...)
	log.Print(dmCmd.String(), "\n")
	return dmCmd.stdout, dmCmd.stderr, dmCmd.err
}

type serviceMap map[string]uint

func machineEnv(machineName string) []string {
	setPrefix := regexp.MustCompile("^SET ")
	envStr, _, err := dockerMachine("env", "--shell", "cmd", machineName) //other shells get extraneous quotes
	if err != nil {
		log.Panic("env", err)
	}

	env := make([]string, 0, 4)
	for _, str := range strings.Split(envStr, "\n") {
		env = append(env, setPrefix.ReplaceAllString(str, ""))
	}

	return env
}

func dockerComposeUp(machineName, dir, ip string, services serviceMap) (upCmd command) {
	upCmd = buildCommand("docker-compose", "up")

	env := machineEnv(machineName)

	upCmd.itself.Env = env
	upCmd.itself.Dir = dir
	upCmd.start()

	if upCmd.err != nil {
		log.Panic(upCmd.err)
	}

	if servicesRunning(60.0, ip, services) {
		return
	}
	panic(fmt.Sprintf("Services were not available!"))
}

func (c *command) interrupt() {
	c.itself.Process.Signal(syscall.SIGTERM)
	c.wait()
}

func dockerComposeDown(cmd *command) error {
	log.Print("Downing compose started by: ", cmd)
	cmd.interrupt()
	if cmd.err != nil {
		return cmd.err
	}

	down := buildCommand("docker-compose", "down")
	down.itself.Env = cmd.itself.Env
	down.itself.Dir = cmd.itself.Dir
	down.run()

	return down.err
}

func servicesRunning(limit float32, ip string, services map[string]uint) bool {
	goodCh := make(chan string)
	badCh := make(chan string)
	done := make(chan bool)
	defer close(done)

	for name, port := range services {
		go func(name, ip string, port uint) {
			if serviceRunning(done, ip, port) {
				goodCh <- name
			} else {
				badCh <- name
			}
		}(name, ip, port)
	}

	for len(services) > 0 {
		select {
		case good := <-goodCh:
			log.Printf("  %s up and running", good)
			delete(services, good)
		case bad := <-badCh:
			log.Printf("  Error trying to connect to %s", bad)
			return false
		case <-time.After(time.Duration(limit * float32(time.Second))):
			log.Printf("Attempt to contact remaining service expired after %f seconds", limit)
			for service, port := range services {
				log.Printf("  Still unavailable: %s at %s:%d", service, ip, port)
			}

			return false
		}
	}
	return true
}

func serviceRunning(done chan bool, ip string, port uint) bool {
	addr := fmt.Sprintf("%s:%d", ip, port)
	log.Print("Attempting connection: ", addr)

	for {
		select {
		case <-done:
			return false
		default:
			conn, err := net.Dial("tcp", addr)
			defer func() {
				if conn != nil {
					conn.Close()
				}
			}()

			if err != nil {
				if _, ok := err.(*net.OpError); ok {
					time.Sleep(time.Duration(0.5 * float32(time.Second)))
					continue
				}
				return false
			}

			return true
		}
	}
}

var ip string
