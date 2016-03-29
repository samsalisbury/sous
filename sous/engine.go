package sous

import "github.com/opentable/sous/util/shell"

type (
	Engine struct {
		MessageHandler func(Message)
	}
)

func (e *Engine) GetSourceContext(sh *shell.Sh) (*BuildContext, error) {
	return nil, nil
}
