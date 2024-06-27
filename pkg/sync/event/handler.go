package event

import (
	"errors"
	"github.com/adobe/cluster-registry/pkg/sqs"
)

type PartialClusterUpdateHandler struct {
	sqs.EventHandler
}

func NewPartialClusterUpdateHandler() *PartialClusterUpdateHandler {
	return &PartialClusterUpdateHandler{}
}

func (h *PartialClusterUpdateHandler) Type() string {
	return sqs.PartialClusterUpdateEvent
}

func (h *PartialClusterUpdateHandler) Handle(event *sqs.Event) error {
	if event == nil {
		return errors.New("event is nil")
	}

	if event.Type != h.Type() {
		return errors.New("event type does not match handler type")
	}

	// TODO

	return nil
}
