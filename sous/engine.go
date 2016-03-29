package sous

type (
	Engine struct {
		MessageHandler func(Message)
	}
)

func (e *Engine) GetSourceContext() (*SourceContext, error) {
	return nil, nil
}
