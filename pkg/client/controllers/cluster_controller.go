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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/sqs"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Queue  sqs.Producer
	CAData string
}

const (
	// HashAnnotation ...
	HashAnnotation = "registry.ethos.adobe.com/hash"
)

//+kubebuilder:rbac:groups=registry.ethos.adobe.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=registry.ethos.adobe.com,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=registry.ethos.adobe.com,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("name", req.NamespacedName)

	instance := new(registryv1.Cluster)
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Error(err, "unable to fetch object")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	skipCACert := instance.Annotations["registry.ethos.adobe.com/skip-ca-cert"]

	// skipCACert is an exception rather than a rule
	if skipCACert != "true" {
		if r.CAData != "" {
			instance.Spec.APIServer.CertificateAuthorityData = r.CAData
		} else {
			log.Info("Certificate Authority data is empty")
		}
	}

	return r.ReconcileCreateUpdate(instance, log)
}

// ReconcileCreateUpdate ...
func (r *ClusterReconciler) ReconcileCreateUpdate(instance *registryv1.Cluster, log logr.Logger) (ctrl.Result, error) {
	hash := hashCluster(instance)

	annotations := instance.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}
	annotations[HashAnnotation] = hash
	instance.SetAnnotations(annotations)

	err := r.enqueue(instance, log)
	if err != nil {
		log.Error(err, "Error enqueuing message")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) hasDifferentHash(object runtime.Object) bool {
	instance := object.(*registryv1.Cluster)
	oldHash := instance.GetAnnotations()[HashAnnotation]
	newHash := hashCluster(instance)

	if oldHash != newHash {
		r.Log.Info("Different hash found", "old", oldHash, "new", newHash,
			"name", fmt.Sprintf("%s/%s", instance.GetNamespace(), instance.GetName()))
		return true
	}
	r.Log.Info("Same hash found", "name", fmt.Sprintf("%s/%s", instance.GetNamespace(), instance.GetName()))
	return false
}

func (r *ClusterReconciler) eventFilters() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			r.Log.Info("CreateEvent", "event", e.Object)
			return r.hasDifferentHash(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			r.Log.Info("UpdateEvent", "event", e.ObjectNew)
			return r.hasDifferentHash(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			r.Log.Info("DeleteEvent", "event", e.Object)
			return !e.DeleteStateUnknown
		},
	}
}

func (r *ClusterReconciler) enqueue(instance *registryv1.Cluster, log logr.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.Queue.Send(ctx, instance)

	if err != nil {
		return err
	}

	log.Info("Successfully enqueued cluster " + instance.Spec.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registryv1.Cluster{}, builder.WithPredicates(r.eventFilters())).
		Complete(r)
}

// hashCluster returns a SHA256 hash of the Cluster object, after removing the ResourceVersion,
// ManagedFields and hashCluster annotation
func hashCluster(instance *registryv1.Cluster) string {
	clone := instance.DeepCopyObject().(*registryv1.Cluster)

	annotations := clone.GetAnnotations()
	delete(annotations, HashAnnotation)
	clone.SetAnnotations(annotations)

	clone.SetResourceVersion("")
	clone.SetManagedFields(nil)

	b, _ := json.Marshal(clone)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", b)))

	return fmt.Sprintf("%x", h.Sum(nil))
}
