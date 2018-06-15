package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

type bootstrapCfg struct {
	repo, offset, flavor, tag string
	serverCfgPath             string
	localListen               string
}

var bootstrap = &cobra.Command{
	Use:   "sous-bootstrap",
	Short: "An operations convenience to bootstrap Sous into execution environments",
	Long: `In normal operations, Sous should be completely self-sufficient.
	  However, if it get's badly broken, or for initial deploys, Sous isn't available
		to deploy itself. sous-bootstrap closes that loop by spinning up a temporary
		local server and issuing deploy commands against that.

		Almost all users should not care this tool exists.
		If you're asking "what's Sous," you're not in the right place, yet.`,
	PreRun: verifyConfig,
	Run:    runBootstrap,
}

var bcfg = bootstrapCfg{}

func init() {
	bootstrap.Flags().StringVarP(&bcfg.serverCfgPath, "server-config", "c", "", "the path to the ephemeral server's config directory")
	bootstrap.Flags().StringVarP(&bcfg.repo, "repo", "r", "", "the repo of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.offset, "offset", "o", "", "the offset of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.flavor, "flavor", "f", "", "the flavor of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.tag, "tag", "t", "", "the tag of the bootstrapped service")
	bootstrap.Flags().StringVarP(&bcfg.localListen, "listen", "l", "localhost:61000", "the address for the ephemeral to listen on")
}

func verifyConfig(*cobra.Command, []string) {
	for _, pair := range []struct{ f, v string }{
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

func runBootstrap(cmd *cobra.Command, args []string) {
	server, err := runServer()
	if err != nil {
		log.Fatal(err)
	}
	defer func(server *exec.Cmd) {
		server.Process.Kill()
		if err := server.Wait(); err != nil {
			log.Fatal(err)
		}
	}(server)

	if err := runDeploy(); err != nil {
		log.Fatal(err)
	}
}

/*
SOUS_SIBLING_URLS='{"$(EMULATE_CLUSTER)": "http://$(LOCAL_SERVER_LISTEN)"}'
SOUS_STATE_LOCATION=$$DIR
SOUS_PG_HOST=$(PGHOST)
SOUS_PG_PORT=$(PGPORT)
SOUS_PG_USER=postgres
*/

func runDeploy() error {
	extraArgs := []string{}
	if bcfg.offset != "" {
		extraArgs = append(extraArgs, "-offset", bcfg.offset)
	}
	if bcfg.flavor != "" {
		extraArgs = append(extraArgs, "-flavor", bcfg.flavor)
	}
	deploy := exec.Command("sous", append([]string{"deploy", "-repo", bcfg.repo, "-tag", bcfg.tag}, extraArgs...)...)
	deploy.Env = os.Environ()
	deploy.Env = append(deploy.Env, "SOUS_SERVER=http://"+bcfg.localListen)
	deploy.Stdout = os.Stdout
	deploy.Stderr = os.Stderr
	return deploy.Run()
}

func runServer() (*exec.Cmd, error) {
	/*
		From the Makefile local-server
		DIR=$(PWD)/.sous-gdm-temp
		rm -rf "$$DIR"
		git clone $(SOUS_GDM_REPO) $$DIR
		sous server -listen $(LOCAL_SERVER_LISTEN) -autoresolver=false
	*/
	server := exec.Command("sous", "server", "-autoresolver=false", "-listen="+bcfg.localListen)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	server.Env = os.Environ()
	server.Env = append(server.Env, "SOUS_CONFIG_DIR="+bcfg.serverCfgPath)

	return server, server.Start()
}
