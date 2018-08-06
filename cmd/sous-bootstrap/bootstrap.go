package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

type bootstrapCfg struct {
	repo, offset, flavor, tag string
	cluster                   string
	serverCfgPath             string
	localListen               string
}

var bootstrap = &cobra.Command{
	Use:   "sous-bootstrap",
	Short: "An operations convenience to bootstrap Sous into execution environments",
	Long: `
In normal operations, Sous should be completely self-sufficient.
However, if it get's badly broken, or for initial deploys, Sous isn't available
to deploy itself. sous-bootstrap closes that loop by spinning up a temporary
local server and issuing deploy commands against that.

Almost all users should not care this tool exists.
If you're asking "what's Sous," you're not in the right place, yet.`,
	PreRun: verifyConfig,
	Run:    wrappedBootstrap,
}

var bcfg = bootstrapCfg{}

func init() {
	bootstrap.Flags().StringVarP(&bcfg.cluster, "cluster", "c", "", "the Sous logical cluster to bootstrap too")
	bootstrap.Flags().StringVarP(&bcfg.serverCfgPath, "server-config", "C", "", "the path to the ephemeral server's config directory")
	bootstrap.Flags().StringVarP(&bcfg.repo, "repo", "r", "", "the repo of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.offset, "offset", "o", "", "the offset of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.flavor, "flavor", "f", "", "the flavor of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.tag, "tag", "t", "", "the tag of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.localListen, "listen", "l", "localhost:61000", "the address for the ephemeral to listen on")
}

func verifyConfig(*cobra.Command, []string) {
	for _, pair := range []struct{ f, v string }{
		{"cluster", bcfg.cluster},
		{"server-config", bcfg.serverCfgPath},
		{"repo", bcfg.repo},
		{"tag", bcfg.tag},
	} {
		if pair.v == "" {
			log.Fatalf("You must provide %s!", pair.f)
		}
	}
	for _, exe := range []string{"sous", "git"} {
		if _, err := exec.LookPath(exe); err != nil {
			log.Fatalf("%q does not appear to be installed, but is required.", exe)
		}
	}
}

func wrappedBootstrap(cmd *cobra.Command, args []string) {
	if err := runBootstrap(); err != nil {
		log.Fatal(err)
	}
}

func runBootstrap() error {
	server, err := runServer()
	if err != nil {
		return err
	}
	log.Printf("Server pid: %d", server.Process.Pid)
	defer func(server *exec.Cmd) {
		server.Process.Kill()
		log.Printf("Waiting for server: %v", server.Wait())
	}(server)

	if err := awaitServer(server); err != nil {
		return err
	}

	return runDeploy()
}

/*
From the Makefile local-server
DIR=$(PWD)/.sous-gdm-temp
rm -rf "$$DIR"
git clone $(SOUS_GDM_REPO) $$DIR
sous server -listen $(LOCAL_SERVER_LISTEN) -autoresolver=false
*/

func runServer() (*exec.Cmd, error) {
	server := exec.Command("sous", "server", "-autoresolver=false", "-listen="+bcfg.localListen, "-cluster="+bcfg.cluster)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	server.Env = os.Environ()
	server.Env = append(server.Env, "SOUS_CONFIG_DIR="+bcfg.serverCfgPath)
	server.Env = append(server.Env, fmt.Sprintf("SOUS_SIBLING_URLS={\"%s\": \"http://%s\"}", bcfg.cluster, bcfg.localListen))

	log.Printf("> sous %s", server.Args)
	return server, server.Start()
}

func awaitServer(server *exec.Cmd) error {
	for i := 0; i < 120; i++ { //30 seconds
		c, err := net.Dial("tcp", bcfg.localListen)
		if err == nil {
			log.Printf("  %s up and running: %v", bcfg.localListen, c)
			return nil
		}
		if _, ok := err.(*net.OpError); !ok {
			log.Print(err)
			return err
		}
		time.Sleep(250 * time.Millisecond)
	}
	err := fmt.Errorf("Server didn't start in 30 seconds.")
	log.Print(err)
	return err
}

func runDeploy() error {
	extraArgs := []string{}
	if bcfg.offset != "" {
		extraArgs = append(extraArgs, "-offset="+bcfg.offset)
	}
	if bcfg.flavor != "" {
		extraArgs = append(extraArgs, "-flavor="+bcfg.flavor)
	}
	deploy := exec.Command("sous", append([]string{"deploy", "-cluster=" + bcfg.cluster, "-repo=" + bcfg.repo, "-tag=" + bcfg.tag}, extraArgs...)...)
	deploy.Env = os.Environ()
	deploy.Env = append(deploy.Env, "SOUS_SERVER=http://"+bcfg.localListen)
	deploy.Stdout = os.Stdout
	deploy.Stderr = os.Stderr
	log.Printf("> sous %s", deploy.Args)
	return deploy.Run()
}
