package publicip

import (
	"context"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type scanner struct {
	client    client.Client
	logger    logr.Logger
	namespace string
}

func (s *scanner) GetClient() client.Client {
	return s.client
}

func (s *scanner) Run(ctx context.Context) error {
	clusterList := &registryv1.ClusterList{}
	err := s.client.List(context.TODO(), clusterList, &client.ListOptions{Namespace: s.namespace})
	if err != nil {
		return err
	}

	for _, cluster := range clusterList.Items {
		switch cluster.Spec.CloudType {
		case "aws", "eks":
			s.logger.Info("Querying AWS cloud provider API", "cluster", cluster.Name)

		case "azure", "aks":
			s.logger.Info("Querying Azure cloud provider API", "cluster", cluster.Name)

		case "datacenter":
			// not yet implemented
			s.logger.Info("Skipping datacenter cluster", "cluster", cluster.Name)

		default:
			s.logger.Info("Unknown cloud provider", "cluster", cluster.Name)
		}
	}

	return nil
}
