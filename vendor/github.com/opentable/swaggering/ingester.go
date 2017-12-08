package swaggering

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type ServiceJSON struct {
	Apis []*ServiceApiJSON
}

type ServiceApiJSON struct {
	Path, Desc string
}

var fileRE = regexp.MustCompile(`^/(\w+).{format}$`)

func ProcessService(dir string, ingester *Context) {
	ingester.ProcessServiceDir(dir)
}

func (ingester *Context) ProcessServiceDir(dir string) {
	fullpath := filepath.Join(dir, "service.json")

	apis := &ServiceJSON{}

	loadJSONfromPath(fullpath, apis)

	for _, api := range apis.Apis {
		sms := fileRE.FindStringSubmatch(api.Path)
		file := string(sms[1])

		ingester.ingestApifromPath(file, filepath.Join(dir, file+".json"))
	}
}

func loadJSONfromPath(fullpath string, into interface{}) {
	data, err := os.Open(fullpath)
	if err != nil {
		log.Print("Trouble with", fullpath, ":", err)
		return
	}

	loadJSON(fullpath, data, into)
}

func loadJSON(fullpath string, data io.Reader, into interface{}) {
	dec := json.NewDecoder(data)

	if err := dec.Decode(into); err == io.EOF {
		log.Fatal("Trouble with empty", fullpath, ":", err)
		return
	} else if err != nil {
		log.Print("Trouble parsing", fullpath, ":", err)
		return
	}
}

func (ctx *Context) ingestApifromPath(name, fullpath string) {
	log.Print("Processing:", fullpath)

	data, err := os.Open(fullpath)
	if err != nil {
		log.Print("Trouble with", fullpath, ":", err)
		return
	}
	ctx.IngestApi(name, fullpath, data)
}

func (ctx *Context) IngestApi(name, aspath string, data io.Reader) {
	ctx.swaggers = append(ctx.swaggers, &Swagger{})
	swagger := ctx.swaggers[len(ctx.swaggers)-1]
	swagger.name = name

	loadJSON(aspath, data, swagger)
}
