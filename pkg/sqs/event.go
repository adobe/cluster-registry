package sqs

import (
	"errors"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"slices"
)

const (
	ClusterUpdateEvent        = "cluster-update"
	PartialClusterUpdateEvent = "partial-cluster-update"
)

type Event struct {
	Type    string
	Message *awssqs.Message
}

func NewEvent(msg *awssqs.Message) (*Event, error) {
	if msg == nil {
		return nil, errors.New("empty message")
	}
	// check if type is set
	if msg.MessageAttributes["Type"] == nil {
		return nil, errors.New("missing event type")
	}
	eventType := *msg.MessageAttributes["Type"].StringValue
	if !slices.Contains([]string{ClusterUpdateEvent, PartialClusterUpdateEvent}, eventType) {
		return nil, errors.New("invalid event type")
	}

	return &Event{
		Type:    eventType,
		Message: msg,
	}, nil
}

type EventHandler interface {
	Type() string
	Handle(event *Event) error
}
