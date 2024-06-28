package sqs

import (
	"errors"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"slices"
)

const (
	MessageAttributeType        = "Type"
	MessageAttributeClusterName = "ClusterName"

	// ClusterUpdateEvent refers to an update of the Cluster object that
	// is sent by the client controller. This event is sent to the SQS queue and
	// is consumed by the API server which reconciles the DB.
	ClusterUpdateEvent = "cluster-update"

	// PartialClusterUpdateEvent refers to an update of the ClusterSync object on the
	// management cluster which is sent by the sync controller. This event is sent to
	// the SQS queue and is consumed by the sync client which creates/updates the
	// Cluster object on the cluster.
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
	if msg.MessageAttributes[MessageAttributeType] == nil {
		return nil, errors.New("missing event type")
	}
	eventType := *msg.MessageAttributes[MessageAttributeType].StringValue
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
