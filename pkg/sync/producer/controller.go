package producer

import (
	"context"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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
	FingerprintAnnotation = "registry.ethos.adobe.com/fingerprint"
)

type SyncController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	WatchedGVKs []schema.GroupVersionKind
}

func (c *SyncController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var start = time.Now()

	log := c.Log.WithValues("name", req.NamespacedName)
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

	if err := c.setFingerprints(ctx, instance); err != nil {
		log.Error(err, "failed to set fingerprints")
		return requeueIfError(err)
	}

	if c.hasDifferentFingerprint(instance) {
		// TODO: reconcile data -> send to SQS
		return noRequeue()
	}

	var errList []error

	for _, res := range instance.Spec.WatchedResources {
		obj := new(unstructured.Unstructured)
		obj.SetAPIVersion(res.APIVersion)
		obj.SetKind(res.Kind)

		err := c.extractMetadata(ctx, obj)
		if err != nil {
			log.Error(err, "failed to extract metadata", "resource", res)
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		return requeueIfError(errList[0])
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
		}))
	}
	err := b.WithOptions(options).Complete(c)
	return err
}

func (c *SyncController) isAllowedGVK(gvk schema.GroupVersionKind) bool {
	for _, watchedGVK := range c.WatchedGVKs {
		if gvk.String() == watchedGVK.String() {
			return true
		}
	}
	return false
}

func (c *SyncController) eventFilters() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e crevent.CreateEvent) bool {
			c.Log.Info("new event", "type", "Create", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return c.hasDifferentFingerprint(e.Object)
		},
		UpdateFunc: func(e crevent.UpdateEvent) bool {
			c.Log.Info("new event", "type", "Update", "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			return c.hasDifferentFingerprint(e.ObjectNew)
		},
		DeleteFunc: func(e crevent.DeleteEvent) bool {
			c.Log.Info("new event", "type", "Delete", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return !e.DeleteStateUnknown
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
					c.Log.Error(err, "failed to parse API version")
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
						c.Log.Error(err, "failed to parse label selector")
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
			}
		}

		return requests
	}
}

func (c *SyncController) extractMetadata(ctx context.Context, obj client.Object) error {
	gvk := obj.GetObjectKind().GroupVersionKind()

	c.Log.Info("extracting metadata", "gvk", gvk.String())

	//switch gvk {
	//case schema.GroupVersionKind{Group: "ec2.services.k8s.aws", Version: "v1alpha1", Kind: "VPC"}:
	//	return c.extractVPCMetadata(ctx, obj)
	//}

	return nil
}

func (c *SyncController) hasDifferentFingerprint(object runtime.Object) bool {
	instance := object.(*registryv1alpha1.ClusterSync)

	oldFingerprint := instance.GetAnnotations()[FingerprintAnnotation]
	newFingerprint := fingerprint(instance.Data)

	if oldFingerprint != newFingerprint {
		c.Log.Info("different fingerprints found",
			"old", oldFingerprint, "new", newFingerprint,
			"namespace", instance.GetNamespace(), "name", instance.GetName())
		return true
	}

	c.Log.Info("same fingerprints found",
		"namespace", instance.GetNamespace(), "name", instance.GetName())
	return false
}

func (c *SyncController) setFingerprints(ctx context.Context, instance *registryv1alpha1.ClusterSync) error {
	annotations := instance.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}
	annotations[FingerprintAnnotation] = fingerprint(instance.Data)
	instance.SetAnnotations(annotations)

	err := c.Client.Update(ctx, instance)
	if err != nil {
		return err
	}

	return nil
}
