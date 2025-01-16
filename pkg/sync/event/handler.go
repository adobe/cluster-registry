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

package event

import (
	"errors"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
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

	log.Info("Handling partial cluster update event")
	log.Info(spew.Sdump(event))

	return nil
}
