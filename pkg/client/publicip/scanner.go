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
