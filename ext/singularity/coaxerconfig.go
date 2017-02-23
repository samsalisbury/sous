package singularity

import (
	"log"
	"time"

	"github.com/opentable/sous/util/coaxer"
)

// c is a temporary global, it will be moved somewhere more sensible soon.
var c = coaxer.NewCoaxer(func(c *coaxer.Coaxer) {
	messages := make(chan string)
	go func() {
		for m := range messages {
			log.Println(m)
		}
	}()
	c.DebugFunc = func(desc string) {
		messages <- desc
	}
	c.Backoff = time.Second
})
