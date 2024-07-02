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

package k8s

import (
	"encoding/base64"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type AzureSPCredentials struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	ResourceID   string
}

type ClientProviderI interface {
	GetClient(appConfig *config.AppConfig, cluster *registryv1.Cluster) (kubernetes.Interface, error)
}

type ClientProvider struct{}

func (cp *ClientProvider) GetClient(appConfig *config.AppConfig, cluster *registryv1.Cluster) (kubernetes.Interface, error) {
	credentials := getCredentials(appConfig)

	cfg, err := clientcmd.BuildConfigFromFlags(cluster.Spec.APIServer.Endpoint, "")
	if err != nil {
		return nil, err
	}

	accessToken, err := getToken(credentials)
	if err != nil {
		return nil, err
	}
	cfg.BearerToken = accessToken

	decodedCAData, err := base64.StdEncoding.DecodeString(cluster.Spec.APIServer.CertificateAuthorityData)
	if err != nil {
		return nil, err
	}
	cfg.CAData = decodedCAData

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getCredentials(appConfig *config.AppConfig) *AzureSPCredentials {
	return &AzureSPCredentials{
		ClientID:     appConfig.ApiClientId,
		ClientSecret: appConfig.ApiClientSecret,
		TenantID:     appConfig.ApiTenantId,
		ResourceID:   appConfig.K8sResourceId,
	}
}

func getToken(credentials *AzureSPCredentials) (string, error) {
	clientCredentials := auth.NewClientCredentialsConfig(credentials.ClientID, credentials.ClientSecret, credentials.TenantID)
	token, err := clientCredentials.ServicePrincipalToken()
	if err != nil {
		return "", err
	}

	err = token.RefreshExchange(credentials.ResourceID)
	if err != nil {
		return "", err
	}

	return token.Token().AccessToken, nil
}
