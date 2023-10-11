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
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	crevent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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

	var patches [][]byte

	for _, wso := range instance.Spec.WatchedServiceObjects {
		gv, err := schema.ParseGroupVersion(wso.ObjectReference.APIVersion)
		if err != nil {
			log.Error(err, "cannot parse APIVersion", "apiVersion", wso.ObjectReference.APIVersion)
			// TODO: update status with error
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
			// TODO: update status with error
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

			value, found, err := getNestedString(obj.Object, strings.Split(field.Source, "."))
			if err != nil {
				// TODO: update status with error
				log.Error(err, "cannot get field", "field", field.Source)
				continue
			}

			if !found {
				log.Error(err, "field not found", "field", field.Source)
				// TODO: update status with error
				continue
			}

			patch, err := createServiceMetadataPatch(serviceId, instance.Namespace, field.Destination, value)
			if err != nil {
				log.Error(err, "cannot create patch")
				continue
			}
			patches = append(patches, patch)
		}
	}

	if len(patches) == 0 {
		return noRequeue()
	}

	if err := r.applyServiceMetadataPatches(ctx, patches); err != nil {
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
		b.Watches(obj, handler.EnqueueRequestsFromMapFunc(r.enqueueRequestsFromMapFunc(gvk)), builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e crevent.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e crevent.UpdateEvent) bool {
				return true
			},
			DeleteFunc: func(e crevent.DeleteEvent) bool {
				// not interested in ServiceObject delete events (for now)
				return false
			},
		}))
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

func (r *ServiceMetadataWatcherReconciler) applyServiceMetadataPatches(ctx context.Context, patches [][]byte) error {
	clusterList := &registryv1.ClusterList{}
	// TODO: get namespace from config
	if err := r.Client.List(ctx, clusterList, &client.ListOptions{Namespace: "cluster-registry"}); err != nil {
		return err
	}

	for i := range clusterList.Items {
		for _, patch := range patches {
			// TODO: find a better way to do this (i.e. group patches)
			if err := r.Client.Patch(ctx, &clusterList.Items[i], client.RawPatch(types.MergePatchType, patch)); err != nil {
				r.Log.Info("cannot patch cluster", "name", clusterList.Items[i].Name, "namespace", clusterList.Items[i].Namespace, "error", err)
			}
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

func createServiceMetadataPatch(serviceId string, namespace string, field string, value string) ([]byte, error) {
	oldCluster := &registryv1.Cluster{}
	oldClusterJSON, err := json.Marshal(oldCluster)
	if err != nil {
		return nil, err
	}

	newCluster := oldCluster.DeepCopy()
	newCluster.Spec.ServiceMetadata = registryv1.ServiceMetadata{
		serviceId: registryv1.ServiceMetadataItem{
			namespace: registryv1.ServiceMetadataMap{
				field: value,
			},
		},
	}
	newClusterJSON, err := json.Marshal(newCluster)
	if err != nil {
		return nil, err
	}

	return jsonpatch.CreateMergePatch(oldClusterJSON, newClusterJSON)
}

// getNestedString returns the value of a nested field in the provided object
// path is a list of keys separated by dots, e.g. "spec.template.spec.containers[0].image"
// if the field is a slice, the last key must be in the form of "key[index]"
func getNestedString(object interface{}, path []string) (string, bool, error) {
	re := regexp.MustCompile(`^(.*)\[(\d+)]$`)
	var cpath []string
	for i, key := range path {
		m := re.FindStringSubmatch(key)
		if len(m) > 0 {
			cpath = append(cpath, m[1])
			slice, found, err := unstructured.NestedSlice(object.(map[string]interface{}), cpath...)
			if !found || err != nil {
				return "", false, err
			}
			index, err := strconv.Atoi(m[2])
			if err != nil && m[2] != "last" {
				return "", false, fmt.Errorf("invalid array index: %s", m[2])
			}
			if m[2] == "last" {
				index = len(slice) - 1
			}
			if len(slice) <= index {
				return "", false, fmt.Errorf("index out of range")
			}
			return getNestedString(slice[index], path[i+1:])
		}
		cpath = append(cpath, key)
	}

	if reflect.TypeOf(object).String() == "string" {
		return object.(string), true, nil
	}

	if reflect.TypeOf(object).String() == "bool" {
		return strconv.FormatBool(object.(bool)), true, nil
	}

	stringVal, found, err := unstructured.NestedString(object.(map[string]interface{}), path...)
	if found && err == nil {
		return stringVal, found, err
	}

	boolVal, found, err := unstructured.NestedBool(object.(map[string]interface{}), path...)
	if found && err == nil {
		return strconv.FormatBool(boolVal), found, err
	}

	// TODO: handle additional types?

	return "", found, fmt.Errorf("invalid field type")
}
