package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	// Setting up logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("sous-server:")

	// Getting config from Mesos ENV
	port0 := os.Getenv("PORT0")
	gdmRepo := os.Getenv("GDM_REPO")

	// Building HTTP config
	if port0 == "" {
		port0 = "80"
	}
	listen := ":" + port0

	log.Printf("Configured:\nListening on %q\nState from %q", listen, gdmRepo)

	// Determining location for GDM git repo
	slConfig := exec.Command("/go/bin/sous", "config", "StateLocation")
	res, err := slConfig.Output()
	if err != nil {
		log.Panic(err)
	}
	stateLocation := strings.TrimSpace(string(res))

	// Checking out GDM
	cloneGDM := exec.Command("git", "clone",
		"--config", "user.email=sous-server@example.com",
		"--config", "user.name=Sous Server",
		gdmRepo, stateLocation)
	log.Printf("Cloning GDM: %#v", cloneGDM)
	err = cloneGDM.Run()
	if err != nil {
		log.Panic(err)
	}

	// Running the `sous server` command proper
	args := []string{"server", "-d", "-v", "-listen", listen}
	sous := exec.Command("/go/bin/sous", args...)
	sous.Stdout = os.Stdout
	sous.Stderr = os.Stderr
	err = sous.Start()
	if err != nil {
		log.Panic(err)
	}

	log.Println("Waiting for sous server to start on " + listen)
	for i := 0; i < 10; i++ {
		r, err := http.Get("http://" + listen)
		if err == nil {
			r.Body.Close()

			// Block until server dies
			sous.Process.Wait()
			return
		}
		time.Sleep(1 * time.Second)
	}
	log.Fatalf("Server did not start within 10s")
}
