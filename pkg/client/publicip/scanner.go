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

package publicip

import (
	"context"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Scanner interface {
	GetClient() client.Client
	Run(ctx context.Context) error
}

func NewScanner(opts ...Option) (Scanner, error) {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}
	options, err := setDefaultOptions(options)
	if err != nil {
		options.Logger.Error(err, "failed to set defaults")
		return nil, err
	}

	return &scanner{
		client:    options.Client,
		logger:    options.Logger,
		namespace: options.Namespace,
	}, nil
}

type Options struct {
	Logger    logr.Logger
	Client    client.Client
	Namespace string
}

type Option func(*Options)

func setDefaultOptions(options Options) (Options, error) {
	if options.Logger.GetSink() == nil {
		options.Logger = log.Log.WithName("publicip-scanner")
	}

	return options, nil
}
