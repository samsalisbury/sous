package smoke

import (
	"encoding/json"
	"log"

	"github.com/opentable/sous/ext/otpl"
	"github.com/opentable/sous/util/filemap"
)

// makeOTPLConfig creates valid otpl-deploy config files using reqID as the
// request ID and envFull as the <env>[.<flavor>] string used by otpl-deploy.
//
// Optionally, pass funcs tweakObjs. These are called with pointers to the
// results of the valid objects being marshalled to JSON and then unmarshalled
// to interface. This gives you a chance to add, set, or remove arbitrary fields
// for testing purposes.
func makeOTPLConfig(reqID, envFull string, tweakObjs ...func(req, dep *interface{})) filemap.FileMap {

	dep := otpl.SingularityJSON{
		RequestID: reqID,
		Resources: otpl.SingularityResources{
			"cpus":     0.01,
			"memoryMb": 1,
			"numPorts": 1,
		},
	}

	req := otpl.SingularityRequestJSON{
		ID:          reqID,
		RequestType: "SERVICE",
		Instances:   1,
		Owners:      []string{"test-user1@example.org"},
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		log.Panicf("Marshalling valid SingularityRequestJSON failed: %s", err)
	}
	depJSON, err := json.Marshal(dep)
	if err != nil {
		log.Panicf("Marshalling valid SingularityJSON failed: %s", err)
	}

	if len(tweakObjs) != 0 {
		var reqObj, depObj interface{}
		if err := json.Unmarshal(reqJSON, &reqObj); err != nil {
			log.Panicf("Unmarshalling SingularityRequestJSON to interface{} failed: %s", err)
		}
		if err := json.Unmarshal(depJSON, &depObj); err != nil {
			log.Panicf("Unmarshalling SingularityJSON to interface{} failed: %s", err)
		}
		for _, f := range tweakObjs {
			f(&reqObj, &depObj)
		}
		var err error
		reqJSON, err = json.Marshal(reqObj)
		if err != nil {
			log.Panicf("Marshalling tweaked SingularityRequestJSON failed: %s", err)
		}
		depJSON, err = json.Marshal(depObj)
		if err != nil {
			log.Panicf("Marshalling tweaked SingularityJSON failed: %s", err)
		}
	}

	return filemap.FileMap{
		"config/" + envFull + "/singularity.json":         string(depJSON),
		"config/" + envFull + "/singularity-request.json": string(reqJSON),
	}
}
