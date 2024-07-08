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

package manager

import (
	"bytes"
	"context"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/adobe/cluster-registry/pkg/sync/parser"
	"github.com/aws/aws-sdk-go/aws"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	crevent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	SyncStatusSuccess = "success"
	SyncStatusFail    = "fail"
)

type SyncController struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	WatchedGVKs    []schema.GroupVersionKind
	Queue          *sqs.Config
	ResourceParser *parser.ResourceParser
}

func (c *SyncController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var start = time.Now()

	log := c.Log.WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("start")
	defer log.Info("end", "duration", time.Since(start))

	instance := new(registryv1alpha1.ClusterSync)
	if err := c.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Error(err, "unable to fetch object")
		return requeueIfError(client.IgnoreNotFound(err))
	}

	if instance.ObjectMeta.DeletionTimestamp != nil {
		return noRequeue()
	}

	// clear sync status
	instance.Status.LastSyncStatus = nil
	instance.Status.LastSyncError = nil

	if c.shouldEnqueueData(instance) {
		instance.Status.SyncedDataHash = ptr.To(hash(instance.Status.SyncedData))
		if err := c.enqueueData(instance); err != nil {
			log.Error(err, "failed to enqueue message")
			if err := c.updateStatus(ctx, instance); err != nil {
				return requeueAfter(10*time.Second, err)
			}
			return noRequeue()
		}
		if err := c.updateStatus(ctx, instance); err != nil {
			return requeueAfter(10*time.Second, err)
		}
		return noRequeue()
	}

	var errList []error
	var finalPatch []byte

	for _, res := range instance.Spec.WatchedResources {
		patch, err := c.ResourceParser.Parse(ctx, res)
		if err != nil {
			log.Error(err, "failed to parse resource", "resource", res)
			errList = append(errList, err)
		}
		finalPatch, err = mergePatches(finalPatch, patch)
		if err != nil {
			log.Error(err, "failed to merge patches")
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		instance.Status.LastSyncStatus = ptr.To(SyncStatusFail)
		// only show the first error
		instance.Status.LastSyncError = ptr.To(errList[0].Error())
		instance.Status.LastSyncTime = &metav1.Time{Time: time.Now()}
		log.Error(errList[0], "failed to sync resources")
		if err := c.updateStatus(ctx, instance); err != nil {
			log.Error(err, "failed to update status")
			return requeueIfError(err)
		}
		return noRequeue()
	}

	err := c.applyPatch(ctx, instance, finalPatch)
	if err != nil {
		log.Error(err, "failed to apply patch")
		return noRequeue()
	}

	return noRequeue()
}

func (c *SyncController) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	options := controller.Options{MaxConcurrentReconciles: 10}
	b := ctrl.NewControllerManagedBy(mgr).For(&registryv1alpha1.ClusterSync{}, builder.WithPredicates(c.eventFilters()))
	for _, gvk := range c.WatchedGVKs {
		obj := new(unstructured.Unstructured)
		obj.SetGroupVersionKind(gvk)
		b.Watches(obj, handler.EnqueueRequestsFromMapFunc(c.enqueueRequestsFromMapFunc(gvk)), builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e crevent.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e crevent.UpdateEvent) bool {
				return true
			},
			DeleteFunc: func(e crevent.DeleteEvent) bool {
				return true
			},
			GenericFunc: func(e crevent.GenericEvent) bool {
				return false
			},
		}))
	}
	err := b.WithOptions(options).Complete(c)
	return err
}

func (c *SyncController) eventFilters() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e crevent.CreateEvent) bool {
			c.Log.Info("new event", "type", "Create", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return c.hasDifferentHash(e.Object)
		},
		UpdateFunc: func(e crevent.UpdateEvent) bool {
			c.Log.Info("new event", "type", "Update", "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())

			oldObject := e.ObjectOld.(*registryv1alpha1.ClusterSync)
			newObject := e.ObjectNew.(*registryv1alpha1.ClusterSync)

			// check if the data has changed
			if c.hasDifferentHash(e.ObjectNew) {
				return true
			}

			// check if the watched resources have changed
			if !reflect.DeepEqual(oldObject.Spec.WatchedResources, newObject.Spec.WatchedResources) {
				return true
			}

			// check if the initial data has changed
			if oldObject.Spec.InitialData != newObject.Spec.InitialData {
				return true
			}

			return false
		},
		DeleteFunc: func(e crevent.DeleteEvent) bool {
			c.Log.Info("new event", "type", "Delete", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return !e.DeleteStateUnknown
		},
		GenericFunc: func(e crevent.GenericEvent) bool {
			c.Log.Info("new event", "type", "Generic", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return false
		},
	}
}

func (c *SyncController) enqueueRequestsFromMapFunc(gvk schema.GroupVersionKind) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		var requests []reconcile.Request

		if obj.GetObjectKind().GroupVersionKind() != gvk {
			// object gvk is of no interest, carry on
			return requests
		}

		// limit the search to the cluster sync namespace
		list := &registryv1alpha1.ClusterSyncList{}
		if err := c.List(ctx, list, &client.ListOptions{Namespace: obj.GetNamespace()}); err != nil {
			c.Log.Error(err, "failed to list ClusterSync objects",
				"namespace", obj.GetNamespace())
			return requests
		}

		for _, clusterSync := range list.Items {
			for _, res := range clusterSync.Spec.WatchedResources {
				gv, err := schema.ParseGroupVersion(res.APIVersion)
				if err != nil {
					c.Log.Error(err, "failed to parseResource API version")
					return requests
				}
				if gv != gvk.GroupVersion() || res.Kind != gvk.Kind || clusterSync.Namespace != obj.GetNamespace() {
					continue
				}
				// if the name is specified, only enqueue if the object name matches
				if res.Name != "" {
					if res.Name != obj.GetName() {
						continue
					}
				}
				// if the label selector is specified, only enqueue if the object labels match
				if res.LabelSelector.Size() > 0 {
					selector, err := metav1.LabelSelectorAsSelector(res.LabelSelector)
					if err != nil {
						c.Log.Error(err, "failed to parseResource label selector")
						return requests
					}
					if !selector.Matches(labels.Set(obj.GetLabels())) {
						continue
					}
				}

				c.Log.Info("watched resource was updated, enqueueing reconcile request",
					"name", obj.GetName(),
					"namespace", obj.GetNamespace(),
					"gvk", gvk.String())

				// enqueue reconcile request for the ClusterSync object
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      clusterSync.GetName(),
						Namespace: clusterSync.GetNamespace(),
					},
				})

				break
			}
		}

		return requests
	}
}

// applyPatch merges the provided patch with the initial data and updates the status of the ClusterSync object
func (c *SyncController) applyPatch(ctx context.Context, instance *registryv1alpha1.ClusterSync, patch []byte) error {
	var initialData map[string]interface{}
	if err := yaml.Unmarshal([]byte(instance.Spec.InitialData), &initialData); err != nil {
		return err
	}

	if initialData == nil {
		initialData = make(map[string]interface{})
	}

	originalJSON, err := json.Marshal(initialData)
	if err != nil {
		return err
	}

	modifiedJSON, err := jsonpatch.MergeMergePatches(originalJSON, patch)
	if err != nil {
		return err
	}

	var modifiedData map[string]interface{}
	if err := json.Unmarshal(modifiedJSON, &modifiedData); err != nil {
		return err
	}

	buf := bytes.Buffer{}
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(modifiedData); err != nil {
		return err
	}
	modifiedYAML := buf.Bytes()

	if reflect.DeepEqual(initialData, modifiedData) {
		c.Log.Info("no data changes detected",
			"name", instance.GetName(),
			"namespace", instance.GetNamespace())
		return nil
	}

	instance.Status.SyncedData = ptr.To(string(modifiedYAML))
	instance.Status.LastSyncStatus = ptr.To(SyncStatusSuccess)
	instance.Status.LastSyncTime = &metav1.Time{Time: time.Now()}

	return c.updateStatus(ctx, instance)
}

func (c *SyncController) enqueueData(instance *registryv1alpha1.ClusterSync) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj, err := yaml.Marshal(instance.Status.SyncedData)
	if err != nil {
		return err
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	start := time.Now()
	err = c.Queue.Enqueue(ctx, []*awssqs.SendMessageBatchRequestEntry{
		{
			Id:           aws.String(id.String()),
			DelaySeconds: aws.Int64(10),
			MessageAttributes: map[string]*awssqs.MessageAttributeValue{
				"Type": {
					DataType:    aws.String("String"),
					StringValue: aws.String(sqs.PartialClusterUpdateEvent),
				},
			},
			MessageBody: aws.String(string(obj)),
		},
	})
	elapsed := float64(time.Since(start)) / float64(time.Second)
	c.Log.Info("Enqueue time", "time", elapsed)

	if err != nil {
		return err
	}

	return nil
}

func (c *SyncController) shouldEnqueueData(instance *registryv1alpha1.ClusterSync) bool {
	if instance.Status.LastSyncStatus != ptr.To(SyncStatusSuccess) {
		return false
	}

	return c.hasDifferentHash(instance)
}

func (c *SyncController) hasDifferentHash(object runtime.Object) bool {
	instance := object.(*registryv1alpha1.ClusterSync)

	oldHash := instance.Status.SyncedDataHash
	newHash := ptr.To(hash(instance.Status.SyncedData))

	return &oldHash != &newHash
}

func (c *SyncController) updateStatus(ctx context.Context, instance *registryv1alpha1.ClusterSync) error {
	if err := c.Client.Status().Update(ctx, instance); err != nil {
		c.Log.Error(err, "failed to update ClusterSync status")
		return err
	}

	c.Log.Info("updated ClusterSync status", "name", instance.GetName(), "namespace", instance.GetNamespace(), "status", instance.Status)
	return nil
}
