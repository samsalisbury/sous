package docker

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
)

func intermediateTag() string {
	return fmt.Sprintf("sousintermediate-%s", uuid.NewV4())
}
