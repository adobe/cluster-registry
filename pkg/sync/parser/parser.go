package parser

import (
	"context"
	"errors"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/adobe/cluster-registry/pkg/sync/parser/handler"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceParser is responsible for parsing any watched resource that has a registered handler
type ResourceParser struct {
	client.Client
	log      logr.Logger
	handlers map[schema.GroupVersionKind]handler.ObjectHandler
}

func New(client client.Client, log logr.Logger) *ResourceParser {
	return &ResourceParser{
		Client:   client,
		log:      log,
		handlers: make(map[schema.GroupVersionKind]handler.ObjectHandler),
	}
}

// RegisterHandlerForGVK registers a handler for a specific GroupVersionKind
func (p *ResourceParser) RegisterHandlerForGVK(gvk schema.GroupVersionKind, parser handler.ObjectHandler) {
	p.handlers[gvk] = parser
}

// Parse parses the watched resource and returns a byte array that represents a patch to be applied to the target resource
func (p *ResourceParser) Parse(ctx context.Context, res registryv1alpha1.WatchedResource) ([]byte, error) {
	gvk, err := res.GVK()
	if err != nil {
		return nil, err
	}

	h, ok := p.handlers[gvk]
	if !ok {
		return nil, errors.New("no handler registered for GVK %s" + gvk.String())
	}

	objects, err := p.getObjectsForResource(ctx, res)
	if err != nil {
		return nil, err
	}

	return h.Handle(ctx, objects)
}

func (p *ResourceParser) getObjectsForResource(ctx context.Context, res registryv1alpha1.WatchedResource) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	gvk, err := res.GVK()
	if err != nil {
		return objects, err
	}

	log := p.log.WithValues("gvk", gvk, "namespace", res.Namespace)

	if res.Name != "" {
		// get a single object
		p.log.WithValues("name", res.Name)
		obj := new(unstructured.Unstructured)
		obj.SetGroupVersionKind(gvk)
		if err := p.Client.Get(ctx, types.NamespacedName{
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
		if err := p.Client.List(ctx, list, listOptions); err != nil {
			p.log.Error(err, "cannot list objects")
		}

		objects = append(objects, list.Items...)
	}
	return objects, nil
}
