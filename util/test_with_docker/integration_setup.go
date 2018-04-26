// test_with_docker provides utilities for using docker-compose for writing
// integration tests.
package test_with_docker

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"
)

type (
	serviceMap map[string]uint
)

var (
	rnums = rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))

	md5RE        = regexp.MustCompile(`(?m)^([0-9a-fA-F]+)\s+(\S+)$`)
	md5missingRE = regexp.MustCompile(`(?m)^md5sum: (?:can't open '(.*)'|(.*)): No such file or directory$`)
	ip           string
)

const (
	// DefaultTimeout is the default timeout for docker operations.
	DefaultTimeout = 30 * time.Second
)

func fileDiffs(pathPairs [][]string, localMD5, remoteMD5 map[string]string) [][]string {
	differentPairs := make([][]string, 0, len(pathPairs))
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	for _, pair := range pathPairs {
		localPath, remotePath := pair[0], pair[1]

		localHash, localPresent := localMD5[localPath]
		remoteHash, remotePresent := remoteMD5[remotePath]

		tw.Write([]byte(fmt.Sprintf("%s\t%t\t%s\n", localPath, localPresent, localHash)))
		tw.Write([]byte(fmt.Sprintf("%s\t%t\t%s\n", remotePath, remotePresent, remoteHash)))

		if localPresent != remotePresent || strings.Compare(remoteHash, localHash) != 0 {
			differentPairs = append(differentPairs, []string{localPath, remotePath})
			log.Printf(" - differ!\n")
		} else {
			log.Printf(" - same.\n")
		}
	}
	tw.Flush()

	return differentPairs
}

func composeService(dir string, ip net.IP, env []string, servicePorts serviceMap, timeout time.Duration) (shutdown *command, err error) {
	if !servicesRunning(3.0*time.Millisecond, ip, servicePorts) {
		log.Printf("Services need to be started - tip: running `docker-compose up` in %s will speed tests up.", dir)

		shutdownCmd := dockerComposeUp(dir, ip, env, servicePorts, timeout)
		shutdown = &shutdownCmd
	} else {
		log.Printf("All services already up and running")
	}
	return
}

func dockerComposeUp(dir string, ip net.IP, env []string, services serviceMap, timeout time.Duration) (upCmd command) {
	log.Println("Pulling compose config in ", dir)
	pullCmd := buildCommand("docker-compose", "pull")
	pullCmd.itself.Env = env
	pullCmd.itself.Dir = dir
	pullCmd.run()
	log.Println(pullCmd.String())
	upCmd = buildCommand("docker-compose", "up", "-d")

	upCmd.itself.Env = env
	upCmd.itself.Dir = dir
	upCmd.run()

	if upCmd.err != nil {
		log.Println(upCmd.stdout)
		log.Println(upCmd.stderr)
		log.Panic(upCmd.err)
	}

	if servicesRunning(timeout, ip, services) {
		return
	}
	log.Println(upCmd.String())

	logCmd := buildCommand("docker-compose", "logs")
	logCmd.itself.Env = env
	logCmd.itself.Dir = dir
	logCmd.start()
	time.Sleep(1 * time.Second)
	logCmd.interrupt()

	log.Println(logCmd.String())

	panic(fmt.Sprintf("Services were not available!"))
}

func dockerComposeDown(cmd *command) error {
	if cmd != nil {
		log.Print("Downing compose started by: ", cmd)
		cmd.interrupt()
		if cmd.err != nil {
			return cmd.err
		}
	}

	down := buildCommand("docker-compose", "down")
	down.itself.Env = cmd.itself.Env
	down.itself.Dir = cmd.itself.Dir
	down.run()

	return down.err
}

func rebuildService(dir, name string, env []string) error {
	cmd := buildCommand("docker-compose", "build", "--no-cache", name)
	cmd.itself.Env = env
	cmd.itself.Dir = dir
	cmd.run()
	if cmd.err != nil {
		log.Print(cmd.stdout)
		log.Print(cmd.stderr)
	}
	return cmd.err
}

func servicesRunning(timeout time.Duration, ip net.IP, services map[string]uint) bool {
	log.Println("servicesRunning func started...", time.Now())
	defer func() { log.Println("servicesRunning func finished...", time.Now()) }()
	var serviceChecks []ReadyFn

	for name, port := range services {
		serviceChecks = append(serviceChecks, serviceReadyFn(name, ip, port))
	}

	err := UntilReady(time.Second/2, timeout, serviceChecks...)
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

func serviceReadyFn(name string, ip net.IP, port uint) ReadyFn {
	return func() (string, func() bool, func()) {
		var conn net.Conn
		var err error
		addr := fmt.Sprintf("%s:%d", ip, port)
		log.Print("Attempting connection: ", addr)

		test := func() bool {
			conn, err = net.Dial("tcp", addr)
			if err != nil {
				if _, ok := err.(*net.OpError); ok {
					return false
				}
				panic(err)
			}
			log.Printf("  %s up and running", addr)
			return true
		}
		teardown := func() {
			if conn != nil {
				conn.Close()
			}
			if err != nil {
				panic(fmt.Errorf("Still unavailable: %s at %s:%d", name, ip, port))
			}
		}

		return fmt.Sprintf("%s at %s:%d", name, ip, port), test, teardown
	}
}

func localMD5s(paths ...string) (md5s map[string]string) {
	md5s = make(map[string]string)

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			log.Print("while MD5ing: ", err)
			continue
		}

		hash := md5.New()
		io.Copy(hash, file)
		md5s[path] = fmt.Sprintf("%x", hash.Sum(nil))
	}
	return
}
