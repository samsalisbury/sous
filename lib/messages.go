package sous

import (
	"fmt"
	"time"
)

type (
	Message interface {
		Time() time.Time
		Sender() string
		Body() string
	}
	// Message is a message sent from Sous Engine
	message struct {
		time       time.Time
		from, body string
	}
	Error   struct{ Message }
	Warning struct{ Message }
	Info    struct{ Message }
	Debug   struct{ Message }
	// Messenger creates and sends messages. It has an internal queue, and tries
	// hard not to block.
	Messenger struct {
		Owner   string
		Queue   chan Message
		Handler func(Message)
	}
)

func (m message) Time() time.Time { return m.time }
func (m message) Sender() string  { return m.from }
func (m message) Body() string    { return m.body }

func NewMessenger(owner string, handler func(Message)) *Messenger {
	q := make(chan Message, 256)
	go func() {
		for m := range q {
			handler(m)
		}
	}()
	return &Messenger{Owner: owner, Queue: q, Handler: handler}
}

func Messagef(from, format string, v ...interface{}) Message {
	return message{time.Now(), from, fmt.Sprintf(format, v...)}
}

func (m *Messenger) Errorf(format string, v ...interface{}) {
	m.Queue <- Error{Messagef(m.Owner, format, v...)}
}
