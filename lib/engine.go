package sous

import "fmt"

type Engine struct {
	SourceHosts []SourceHost
}

func (e *Engine) ParseSourceLocation(s string) (SourceLocation, error) {
	for _, h := range e.SourceHosts {
		if h.CanParseSourceLocation(s) {
			return h.ParseSourceLocation(s)
		}
	}
	return SourceLocation{}, fmt.Errorf("source location not recognised: %q", s)
}
