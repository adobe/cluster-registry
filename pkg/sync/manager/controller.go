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
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/adobe/cluster-registry/pkg/sqs"
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
	"k8s.io/utils/pointer"
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
	Log         logr.Logger
	Scheme      *runtime.Scheme
	WatchedGVKs []schema.GroupVersionKind
	Queue       *sqs.Config
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
		instance.Status.SyncedDataHash = pointer.String(hash(instance.Status.SyncedData))
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
		patch, err := c.parseResource(ctx, res)
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
		instance.Status.LastSyncStatus = pointer.String(SyncStatusFail)
		// only show the first error
		instance.Status.LastSyncError = pointer.String(errList[0].Error())
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

func (c *SyncController) parseResource(ctx context.Context, res registryv1alpha1.WatchedResource) ([]byte, error) {
	gvk, err := res.GVK()
	if err != nil {
		return nil, err
	}

	switch gvk {

	case schema.GroupVersionKind{Group: "ec2.services.k8s.aws", Version: "v1alpha1", Kind: "VPC"}:
		patch, err := c.extractVPCMetadata(ctx, res)
		if err != nil {
			return nil, err
		}
		return patch, nil

	case schema.GroupVersionKind{Group: "cluster.x-k8s.io", Version: "v1beta1", Kind: "Cluster"}:
		patch, err := c.extractClusterMetadata(ctx, res)
		if err != nil {
			return nil, err
		}
		return patch, nil
	}

	return nil, nil
}

func (c *SyncController) extractObjectsForResource(ctx context.Context, res registryv1alpha1.WatchedResource) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	gvk, err := res.GVK()
	if err != nil {
		return objects, err
	}

	log := c.Log.WithValues("gvk", gvk, "namespace", res.Namespace)

	if res.Name != "" {
		// get a single object
		log.WithValues("name", res.Name)
		obj := new(unstructured.Unstructured)
		obj.SetGroupVersionKind(gvk)
		if err := c.Client.Get(ctx, types.NamespacedName{
			Name:      res.Name,
			Namespace: res.Namespace,
		}, obj); err != nil {
			log.Error(err, "cannot get object")
			return nil, err
		}

		objects = append(objects, *obj)
	} else {
		// get a list of objects
		listOptions := &client.ListOptions{Namespace: res.Namespace}
		if res.LabelSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(res.LabelSelector)
			if err != nil {
				return nil, err
			}
			listOptions.LabelSelector = selector
			log.WithValues("labelSelector", res.LabelSelector.String())
		}
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := c.Client.List(ctx, list, listOptions); err != nil {
			c.Log.Error(err, "cannot list objects")

		}

		objects = append(objects, list.Items...)
	}
	return objects, nil
}

func (c *SyncController) extractClusterMetadata(ctx context.Context, res registryv1alpha1.WatchedResource) ([]byte, error) {
	objects, err := c.extractObjectsForResource(ctx, res)
	if err != nil {
		return nil, err
	}

	clusterSpec := new(v1.ClusterSpec)

	for _, obj := range objects {
		clusterShortName, err := getNestedString(obj, "metadata", "labels", "shortName")
		if err != nil {
			return nil, err
		}
		clusterSpec.ShortName = clusterShortName

		region, err := getNestedString(obj, "metadata", "labels", "locationShortName")
		if err != nil {
			return nil, err
		}
		clusterSpec.Region = region

		provider, err := getNestedString(obj, "metadata", "labels", "provider")
		if err != nil {
			return nil, err
		}
		clusterSpec.CloudType = provider

		cloudProviderRegion, err := getNestedString(obj, "metadata", "labels", "location")
		if err != nil {
			return nil, err
		}
		clusterSpec.CloudProviderRegion = cloudProviderRegion

		environment, err := getNestedString(obj, "metadata", "labels", "environment")
		if err != nil {
			return nil, err
		}
		clusterSpec.Environment = environment

	}

	patch, err := createClusterSpecPatch(clusterSpec)
	if err != nil {
		return nil, err
	}

	c.Log.Info("extracted Cluster metadata", "patch", string(patch))

	return patch, nil
}

func (c *SyncController) extractVPCMetadata(ctx context.Context, res registryv1alpha1.WatchedResource) ([]byte, error) {
	objects, err := c.extractObjectsForResource(ctx, res)
	if err != nil {
		return nil, err
	}

	clusterSpec := new(v1.ClusterSpec)

	for _, obj := range objects {
		vpcID, err := getNestedString(obj, "status", "vpcID")
		if err != nil {
			return nil, err
		}

		cidrs, err := getNestedStringSlice(obj, "spec", "cidrBlocks")
		if err != nil {
			return nil, err
		}

		clusterSpec.VirtualNetworks = append(clusterSpec.VirtualNetworks, v1.VirtualNetwork{
			ID:    vpcID,
			Cidrs: cidrs,
		})
	}

	patch, err := createClusterSpecPatch(clusterSpec)
	if err != nil {
		return nil, err
	}

	c.Log.Info("extracted VPC metadata", "patch", string(patch))

	return patch, nil
}

// TODO: think of a better way to do this
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

	instance.Status.SyncedData = pointer.String(string(modifiedYAML))
	instance.Status.LastSyncStatus = pointer.String(SyncStatusSuccess)
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
	if instance.Status.LastSyncStatus != pointer.String(SyncStatusSuccess) {
		return false
	}

	return c.hasDifferentHash(instance)
}

func (c *SyncController) hasDifferentHash(object runtime.Object) bool {
	instance := object.(*registryv1alpha1.ClusterSync)

	oldHash := instance.Status.SyncedDataHash
	newHash := pointer.String(hash(instance.Status.SyncedData))

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
