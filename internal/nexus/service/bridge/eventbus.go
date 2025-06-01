package bridge

import "context"

type Event struct {
	Type        string
	ID          string
	Source      string
	Destination string
	Metadata    map[string]string
	Payload     []byte
	Timestamp   int64
}

type ErrorEvent struct {
	Error   string
	Message *Message
}

type EventBus interface {
	Subscribe(topic string, handler func(context.Context, *Event)) error
	Publish(topic string, event *Event) error
}
