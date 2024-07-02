/*
Copyright 2024 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

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
