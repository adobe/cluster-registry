package manager

import (
	"bytes"
	"context"
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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

	if err := c.setFingerprints(ctx, instance); err != nil {
		log.Error(err, "failed to set fingerprints")
		return requeueIfError(err)
	}

	if c.hasDifferentFingerprint(instance) {
		// TODO: reconcile data -> send to SQS

		c.Log.Info("reconciling data", "data", instance.Data)

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

			oldObject := e.ObjectOld.(*registryv1alpha1.ClusterSync)
			newObject := e.ObjectNew.(*registryv1alpha1.ClusterSync)

			// check if the data has changed
			if c.hasDifferentFingerprint(e.ObjectNew) {
				return true
			}

			// check if the watched resources have changed
			if !reflect.DeepEqual(oldObject.Spec.WatchedResources, newObject.Spec.WatchedResources) {
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
		clusterShortName, err := getNestedString(obj, "metadata", "labels", "clusterShortName")
		if err != nil {
			c.errorFailedToGetValueFromObject(err, obj)
			continue
		}
		clusterSpec.ShortName = clusterShortName

		region, err := getNestedString(obj, "metadata", "labels", "locationShortName")
		if err != nil {
			c.errorFailedToGetValueFromObject(err, obj)
			continue
		}
		clusterSpec.Region = region

		provider, err := getNestedString(obj, "metadata", "labels", "provider")
		if err != nil {
			c.errorFailedToGetValueFromObject(err, obj)
			continue
		}
		clusterSpec.CloudType = provider

		cloudProviderRegion, err := getNestedString(obj, "metadata", "labels", "location")
		if err != nil {
			c.errorFailedToGetValueFromObject(err, obj)
			continue
		}
		clusterSpec.CloudProviderRegion = cloudProviderRegion

		environment, err := getNestedString(obj, "metadata", "labels", "environment")
		if err != nil {
			c.errorFailedToGetValueFromObject(err, obj)
			continue
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
			c.errorFailedToGetValueFromObject(err, obj)
			continue
		}

		cidrs, err := getNestedStringSlice(obj, "spec", "cidrBlocks")
		if err != nil {
			c.errorFailedToGetValueFromObject(err, obj)
			continue
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
	var originalData map[string]interface{}
	if err := yaml.Unmarshal([]byte(instance.Data), &originalData); err != nil {
		return err
	}

	originalJSON, err := json.Marshal(originalData)
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

	if reflect.DeepEqual(originalData, modifiedData) {
		c.Log.Info("no data changes detected",
			"name", instance.GetName(),
			"namespace", instance.GetNamespace())
		return nil
	}

	instance.Data = string(modifiedYAML)

	if err := c.Client.Update(ctx, instance); err != nil {
		c.Log.Error(err, "failed to update ClusterSync data")
		return err
	}

	c.Log.Info("updated ClusterSync data",
		"name", instance.GetName(),
		"namespace", instance.GetNamespace(),
		"data", modifiedData)

	return nil
}

// --

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

func (c *SyncController) errorFailedToGetValueFromObject(err error, obj unstructured.Unstructured) {
	c.Log.Info("failed to get value from object", "name", obj.GetName(), "namespace", obj.GetNamespace(), "err", err)
}
