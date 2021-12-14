/*
Copyright 2021 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package e2e

import (
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func getToken() string {
	resourceID := os.Getenv("OIDC_CLIENT_ID")       // Cluster Registry App ID
	tenantID := os.Getenv("OIDC_TENANT_ID")         // Tenant ID
	clientID := os.Getenv("TEST_CLIENT_ID")         // Test App ID
	clientSecret := os.Getenv("TEST_CLIENT_SECRET") // Test App Secret

	clientCredentials := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)

	token, err := clientCredentials.ServicePrincipalToken()
	if err != nil {
		fmt.Println(err)
	}

	err = token.RefreshExchange(resourceID)
	if err != nil {
		fmt.Println(err)
	}

	return token.Token().AccessToken
}
