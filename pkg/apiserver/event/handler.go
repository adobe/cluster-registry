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
	"encoding/json"
	"errors"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/database"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/labstack/gommon/log"
	"strconv"
	"time"
)

type ClusterUpdateHandler struct {
	sqs.EventHandler
	db database.Db
}

func NewClusterUpdateHandler(db database.Db) *ClusterUpdateHandler {
	return &ClusterUpdateHandler{
		db: db,
	}
}

func (h *ClusterUpdateHandler) Type() string {
	return sqs.ClusterUpdateEvent
}

func (h *ClusterUpdateHandler) Handle(event *sqs.Event) error {
	if event == nil {
		return errors.New("event is nil")
	}

	if event.Type != h.Type() {
		return errors.New("event type does not match handler type")
	}

	var rcvCluster registryv1.Cluster

	msg := event.Message

	err := json.Unmarshal([]byte(*msg.Body), &rcvCluster)
	if err != nil {
		log.Error("Failed to unmarshal message.")
		return err
	}

	clusterName := rcvCluster.Spec.Name

	msgTimestamp, err := strconv.ParseInt(*msg.Attributes["SentTimestamp"], 10, 64)
	if err != nil {
		log.Error("Wrong time format for sqs message:", msg.MessageId)
		return err
	}
	lastUpdated := time.Unix(0, msgTimestamp*int64(time.Millisecond))

	cluster, err := h.db.GetCluster(clusterName)
	if err != nil {
		log.Error("Failed to get cluster ", clusterName, " from database.")
		return err
	}

	if cluster == nil {
		rcvCluster.Spec.LastUpdated = lastUpdated.UTC().Format(time.RFC3339Nano)
		err = h.db.PutCluster(&rcvCluster)
		if err != nil {
			log.Error("Cluster ", clusterName, " failed to be created.")
			return err
		}
		log.Info("Cluster ", clusterName, " was created.")
		return nil
	}

	clusterTime, err := time.Parse(time.RFC3339Nano, cluster.Spec.LastUpdated)
	if err != nil {
		log.Warn("Wrong time format in database for: ", clusterName)
	} else if lastUpdated.Before(clusterTime) {
		log.Info("Cluster lastUpdated timestamp is too old. This update will be skip for ", clusterName)
		return nil
	}

	rcvCluster.Spec.LastUpdated = lastUpdated.UTC().Format(time.RFC3339Nano)
	err = h.db.PutCluster(&rcvCluster)
	if err != nil {
		log.Error("Cluster ", clusterName, " failed to be updated.")
		return err
	}

	log.Info("Cluster ", clusterName, " was updated.")
	return err
}
