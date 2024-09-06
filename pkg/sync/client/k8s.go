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

package client

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func getClientSet() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}

func getDynamicClientSet() (*dynamic.DynamicClient, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	dnc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return dnc, nil
}
