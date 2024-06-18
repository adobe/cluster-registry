package producer

import (
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Patcher struct {
	log       logr.Logger
	queue     workqueue.RateLimitingInterface
	indexer   cache.Indexer
	batchSize int
	stopCh    chan struct{}
}

type Patch struct {
	TargetConfigMap types.NamespacedName
	Data            []byte
}

func (p *Patch) String() string {
	return string(p.Data)
}

var PatchKeyFunc = func(obj interface{}) (string, error) {
	patch, ok := obj.(*Patch)
	if !ok {
		return "", fmt.Errorf("object is not of type Patch")
	}
	return patch.TargetConfigMap.String(), nil
}

func NewPatcher(queue workqueue.RateLimitingInterface, indexer cache.Indexer, logger logr.Logger, batchSize int) *Patcher {
	return &Patcher{
		queue:     queue,
		indexer:   indexer,
		batchSize: batchSize,
		log:       logger,
	}
}

func (p *Patcher) AddPatch(patch *Patch) {
	err := p.indexer.Add(patch)
	if err != nil {
		p.log.Error(err, "Error adding patch to indexer")
		return
	}
	key, err := PatchKeyFunc(patch)
	if err != nil {
		p.log.Error(err, "Error getting key for patch")
		return
	}
	p.queue.Add(key)
}

func (p *Patcher) processNextItem() bool {
	key, quit := p.queue.Get()
	if quit {
		return false
	}
	defer p.queue.Done(key)

	patch, exists, err := p.indexer.GetByKey(key.(string))
	if err != nil {
		p.log.Error(err, fmt.Sprintf("Fetching object with key %s from store", key))
	}
	if !exists {
		p.log.Info("Patch not found in store", "key", key)
		return true

	}

	if err := p.handlePatch(patch.(*Patch)); err != nil {
		p.log.Error(err, "Error handling patch", "key", key)
		p.queue.AddRateLimited(key)
	} else {
		p.queue.Forget(key)
	}

	return true
}

func (p *Patcher) handlePatch(patch *Patch) error {
	p.log.Info("Handling patch", "patch", patch)

	return nil
}

func (p *Patcher) processBatch() bool {
	var keys []interface{}
	var patches []*Patch

	for i := 0; i < p.batchSize; i++ {
		key, quit := p.queue.Get()
		if quit {
			return false
		}
		defer p.queue.Done(key)
		keys = append(keys, key)

		patch, exists, err := p.indexer.GetByKey(key.(string))
		if err != nil {
			p.log.Error(err, fmt.Sprintf("Fetching object with key %s from store", key))
			return true
		}
		if !exists {
			p.log.Info("Patch not found in store", "key", key)
			return true
		}
		patches = append(patches, patch.(*Patch))
	}

	if err := p.handleBatch(patches); err != nil {
		p.log.Error(err, "Error handling batch", "keys", keys)
		for _, key := range keys {
			p.queue.AddRateLimited(key)
		}
	} else {
		for _, key := range keys {
			p.queue.Forget(key)
		}
	}

	return true
}

func (p *Patcher) handleBatch(patches []*Patch) error {
	p.log.Info("Handling batch", "patches", patches)

	return nil
}

func (p *Patcher) Run() {
	for {
		select {
		case <-p.stopCh:
			p.log.Info("Stopping patcher")
			return
		default:
			p.processBatch()
		}
	}
}

func (p *Patcher) runWorker() {
	//for p.processNextItem() {
	//}
}
