package docker_images

import (
	"log"
	"net/http"

	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"golang.org/x/net/context"
)

func main() {
	ctx := context.Background()
	name, err := reference.ParseNamed("hello-world")
	if err != nil {
		log.Panic(err)
	}

	baseUrl := "https://otdocker.io"
	xport := new(http.Transport)
	rep, err := client.NewRepository(ctx, name, baseUrl, xport)
	if err != nil {
		log.Panic(err)
	}

	manifests, err := rep.Manifests(ctx)
	if err != nil {
		log.Panic(err)
	}
}
