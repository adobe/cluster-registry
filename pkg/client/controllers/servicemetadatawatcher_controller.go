/*
Copyright 2021 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	crevent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceMetadataWatcherReconciler reconciles a ServiceMetadataWatcher object
type ServiceMetadataWatcherReconciler struct {
	client.Client
	Log                 logr.Logger
	Scheme              *runtime.Scheme
	WatchedGVKs         []schema.GroupVersionKind
	ServiceIdAnnotation string
}

//+kubebuilder:rbac:groups=registry.ethos.adobe.com,resources=servicemetadatawatchers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=registry.ethos.adobe.com,resources=servicemetadatawatchers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=registry.ethos.adobe.com,resources=servicemetadatawatchers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ServiceMetadataWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var start = time.Now()

	log := r.Log.WithValues("name", req.NamespacedName)
	log.Info("start")
	defer log.Info("end", "duration", time.Since(start))

	instance := new(registryv1alpha1.ServiceMetadataWatcher)
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Error(err, "unable to fetch object")
		return requeueIfError(client.IgnoreNotFound(err))
	}

	if instance.ObjectMeta.DeletionTimestamp != nil {
		return noRequeue()
	}

	if len(instance.Spec.WatchedServiceObjects) == 0 {
		log.Info("no watched objects")
		// TODO: probably update status with error
		return noRequeue()
	}

	serviceId, err := r.getServiceIdFromNamespaceAnnotation(ctx, instance.GetNamespace())
	if err != nil {
		log.Error(err, "cannot get serviceId from namespace annotation", "namespace", instance.GetNamespace())
		return noRequeue()
	}

	var updated bool = false
	serviceMetadata := registryv1.ServiceMetadata{
		serviceId: registryv1.ServiceMetadataItem{
			instance.GetNamespace(): registryv1.ServiceMetadataMap{},
		},
	}

	for _, wso := range instance.Spec.WatchedServiceObjects {
		gv, err := schema.ParseGroupVersion(wso.ObjectReference.APIVersion)
		if err != nil {
			log.Error(err, "cannot parse APIVersion", "apiVersion", wso.ObjectReference.APIVersion)
			return requeueIfError(err)
		}

		gvk := schema.GroupVersionKind{
			Group:   gv.Group,
			Version: gv.Version,
			Kind:    wso.ObjectReference.Kind,
		}

		// check if gvk is allowed
		if !r.isAllowedGVK(gvk) {
			log.Info("watched object GVK is not allowed, skipping", "gvk", gvk.String())
			return noRequeue()
		}

		obj := new(unstructured.Unstructured)
		obj.SetGroupVersionKind(gvk)

		if err := r.Client.Get(ctx, types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      wso.ObjectReference.Name,
		}, obj); err != nil {
			log.Error(err, "cannot get object",
				"name", wso.ObjectReference.Name,
				"namespace", instance.Namespace,
				"gvk", obj.GroupVersionKind().String())
			return requeueIfError(err)
		}

		if len(wso.WatchedFields) == 0 {
			// TODO: update status with error
			return noRequeue()
		}

		for _, field := range wso.WatchedFields {
			// TODO: validate field Source & Destination somewhere

			value, found, err := unstructured.NestedString(obj.Object, strings.Split(field.Source, ".")...)
			if err != nil {
				// TODO: update status with error
				return requeueIfError(err)
			}

			if !found {
				// TODO: update status with error
				return requeueIfError(err)
			}

			serviceMetadata[serviceId][obj.GetNamespace()][field.Destination] = value
			updated = true
		}
	}

	if !updated {
		return noRequeue()
	}

	if err := r.updateClusterServiceMetadata(ctx, serviceMetadata); err != nil {
		log.Error(err, "cannot update cluster service metadata")
		return requeueIfError(err)
	}

	return noRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceMetadataWatcherReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	options := controller.Options{MaxConcurrentReconciles: 10}
	b := ctrl.NewControllerManagedBy(mgr).For(&registryv1alpha1.ServiceMetadataWatcher{}, builder.WithPredicates(r.eventFilters()))
	for _, gvk := range r.WatchedGVKs {
		obj := new(unstructured.Unstructured)
		obj.SetGroupVersionKind(gvk)
		b.Watches(obj, handler.EnqueueRequestsFromMapFunc(r.enqueueRequestsFromMapFunc(gvk)))
	}
	err := b.WithOptions(options).Complete(r)
	return err
}

func (r *ServiceMetadataWatcherReconciler) isAllowedGVK(gvk schema.GroupVersionKind) bool {
	for _, watchedGVK := range r.WatchedGVKs {
		if gvk.String() == watchedGVK.String() {
			return true
		}
	}
	return false
}

func (r *ServiceMetadataWatcherReconciler) updateClusterServiceMetadata(ctx context.Context, serviceMetadata registryv1.ServiceMetadata) error {
	clusterList := &registryv1.ClusterList{}
	// TODO: get namespace from config
	if err := r.Client.List(ctx, clusterList, &client.ListOptions{Namespace: "cluster-registry"}); err != nil {
		return err
	}

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]
		newCluster := cluster.DeepCopy()
		newCluster.Spec.ServiceMetadata = serviceMetadata
		patch := client.MergeFrom(cluster)
		rawPatch, err := patch.Data(newCluster)
		if err != nil {
			return err
		} else if string(rawPatch) == "{}" {
			return nil
		}
		if err := r.Client.Patch(ctx, &clusterList.Items[i], client.RawPatch(patch.Type(), rawPatch)); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceMetadataWatcherReconciler) eventFilters() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e crevent.CreateEvent) bool {
			r.Log.Info("New event", "type", "Create", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		},
		UpdateFunc: func(e crevent.UpdateEvent) bool {
			r.Log.Info("New event", "type", "Update", "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			return true
		},
		DeleteFunc: func(e crevent.DeleteEvent) bool {
			r.Log.Info("New event", "type", "Delete", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return !e.DeleteStateUnknown
		},
	}
}

// enqueueRequestsFromMapFunc enqueues reconcile requests for ServiceMetadataWatchers when watched objects are updated
func (r *ServiceMetadataWatcherReconciler) enqueueRequestsFromMapFunc(gvk schema.GroupVersionKind) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		var requests []reconcile.Request

		if obj.GetObjectKind().GroupVersionKind() != gvk {
			// object gvk is of no interest, carry on
			return requests
		}

		// limit the search to the watcher's namespace
		list := &registryv1alpha1.ServiceMetadataWatcherList{}
		if err := r.List(ctx, list, &client.ListOptions{Namespace: obj.GetNamespace()}); err != nil {
			r.Log.Error(err, "failed to list ServiceMetadataWatchers",
				"namespace", obj.GetNamespace())
			return requests
		}

		for _, smw := range list.Items {
			for _, wso := range smw.Spec.WatchedServiceObjects {
				gv, err := schema.ParseGroupVersion(wso.ObjectReference.APIVersion)
				if err != nil || gv != gvk.GroupVersion() || wso.ObjectReference.Kind != gvk.Kind {
					continue
				}
				if smw.Namespace == obj.GetNamespace() && wso.ObjectReference.Name == obj.GetName() {
					r.Log.Info("Watched object was updated, enqueueing watcher reconcile request",
						"name", obj.GetName(),
						"namespace", smw.Namespace,
						"gvk", gvk.String(),
						"watcher", smw.Name)
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: smw.Namespace,
							Name:      smw.Name,
						},
					})
				}
			}
		}

		return requests
	}
}

func (r *ServiceMetadataWatcherReconciler) getServiceIdFromNamespaceAnnotation(ctx context.Context, namespace string) (string, error) {
	ns := &corev1.Namespace{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		r.Log.Error(err, "cannot get namespace", "namespace", namespace)
		return "", err
	}

	serviceId, ok := ns.GetAnnotations()[r.ServiceIdAnnotation]

	if !ok {
		return "", fmt.Errorf("namespace %s does not have annotation %s", namespace, r.ServiceIdAnnotation)
	}

	return serviceId, nil
}
