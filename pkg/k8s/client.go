/*
Copyright 2023 Adobe. All rights reserved.
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type AzureSPCredentials struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	ResourceID   string
}

func Client(cluster *registryv1.Cluster, credentials AzureSPCredentials) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags(cluster.Spec.APIServer.Endpoint, "")
	if err != nil {
		return nil, err
	}

	accessToken, err := getToken(credentials)
	if err != nil {
		return nil, err
	}
	config.BearerToken = accessToken

	decodedCAData, err := base64.StdEncoding.DecodeString(cluster.Spec.APIServer.CertificateAuthorityData)
	if err != nil {
		return nil, err
	}
	config.CAData = decodedCAData

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getToken(credentials AzureSPCredentials) (string, error) {
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
