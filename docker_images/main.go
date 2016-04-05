package main

import (
	"log"
	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	_ "github.com/docker/distribution/manifest/schema1"
	_ "github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"golang.org/x/net/context"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	baseUrl := "http://artifactory.otenv.com/artifactory/api/docker/docker-v2/v2"
	repstr := "demo-server"
	tagstr := "demo-server-0.7.3-SNAPSHOT-20160329_202654_teamcity-unconfigured"

	ctx := context.Background()
	name, err := reference.ParseNamed(repstr)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("%+v", name)

	xport := new(http.Transport)
	rep, err := client.NewRepository(ctx, name, baseUrl, xport)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("%+v", rep)

	manifests, err := rep.Manifests(ctx)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("%+v", manifests)

	mani, err := manifests.Get(ctx, digest.Digest(""), distribution.WithTagOption{Tag: tagstr})
	if err != nil {
		log.Panic(err)
	}

	log.Printf("%+v", mani)

}
