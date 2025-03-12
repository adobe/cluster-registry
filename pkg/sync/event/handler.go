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
	"context"
	"errors"
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/adobe/cluster-registry/pkg/sync/client"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
	"strconv"
)

type PartialClusterUpdateHandler struct {
	sqs.EventHandler
	client    *dynamic.DynamicClient
	namespace string
}

func NewPartialClusterUpdateHandler(
	client *dynamic.DynamicClient,
	namespace string,
) *PartialClusterUpdateHandler {
	return &PartialClusterUpdateHandler{client: client, namespace: namespace}
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

	// try to get cluster name from message
	clusterName, err := event.GetClusterName()
	if err != nil {
		log.Error("Failed to get cluster name from message")
		return err
	}

	msg, _ := strconv.Unquote(*event.Message.Body)

	data, err := client.PartialClusterMergePatch([]byte(msg))
	if err != nil {
		log.Error("Failed to create partial merge patch: ", err)
		return err
	}

	clusterResource := v1.GroupVersion.WithResource("clusters")

	// try to patch Cluster object if it exists
	_, err = h.client.Resource(clusterResource).Namespace(h.namespace).Patch(context.TODO(), clusterName, types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil {

		// if Cluster object does not exist, attempt to create it
		if kerrors.IsNotFound(err) {
			log.Info("Cluster object not found, checking if it is a new cluster")

			clusterSpec := v1.ClusterSpec{}
			err = json.Unmarshal([]byte(msg), &clusterSpec)
			if err != nil {
				log.Error("Failed to unmarshal cluster spec from message")
				return err
			}

			cluster := v1.Cluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Cluster",
					APIVersion: v1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: h.namespace,
				},
				Spec: clusterSpec,
			}

			obj, err := client.ToUnstructured(cluster)
			if err != nil {
				log.Error("Failed to convert cluster to unstructured.")
			}

			create, err := h.client.Resource(clusterResource).Namespace(h.namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
			if err != nil {
				// if Cluster object is invalid, log and return;
				// this most likely means that the sync-manager has not yet gathered all the required data for this cluster
				if kerrors.IsInvalid(err) {
					log.Error("Invalid Cluster object: ", err)
					return nil
				}
				log.Error("Failed to create Cluster object: ", err)
				return err
			}
			log.Info("Cluster object created: ", create.GetName())
		}
		return err
	}

	return nil
}
