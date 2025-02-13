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
	"crypto/sha256"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/time/rate"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func hash(obj interface{}) string {
	b, _ := json.Marshal(obj)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", b)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func NewControllerRateLimiter(
	exponentialFailureBaseDelay time.Duration,
	exponentialFailureMaxDelay time.Duration,
	overallBucketQPS int,
	overallBucketBurst int) workqueue.TypedRateLimiter[reconcile.Request] {
	return workqueue.NewTypedMaxOfRateLimiter[reconcile.Request](
		workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](
			exponentialFailureBaseDelay,
			exponentialFailureMaxDelay),
		&workqueue.TypedBucketRateLimiter[reconcile.Request]{
			Limiter: rate.NewLimiter(rate.Limit(overallBucketQPS), overallBucketBurst)},
	)
}
