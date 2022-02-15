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

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	"github.com/adobe/cluster-registry/test/jwt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	jose "gopkg.in/square/go-jose.v2"
)

const (
	noSignatureToken = `
	eyJhbGciOiJSUzI1NiIsImtpZCI6ImNkNTVlNTFiODM3YmMxM2Q4NzNjZmYxYTllY2ZmZTIyOTlkMTE1ZTAyOTUwYTM2ZTNiZDY2ZTVmZTBlNzNmNTYifQ.eyJhdWQiOiJvaWRjLWNsaWVudC1pZCIsImV4cCI6IjE2NDIwMjQxMzkiLCJpYXQiOiIxNjQyMDIwNTM5IiwiaXBkIjoiaHR0cDovL2Zha2Utb2lkYy1wcm92aWRlciIsImlzcyI6Imh0dHA6Ly9mYWtlLW9pZGMtcHJvdmlkZXIiLCJvaWQiOiIwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDAifQ
`
	signingKeyPrivate          = "RSA PRIVATE KEY"
	signingKeyPublic           = "RSA PUBLIC KEY"
	dummySigningKeyFile        = "../../../test/testdata/dummyRsaPrivateKey.pem"
	invalidDummySigningKeyFile = "../../../test/testdata/invalidDummyRsaPrivateKey.pem"
)

// staticKeySet implements oidc.KeySet
type staticKeySet struct {
	keys []*jose.JSONWebKey
}

// VerifySignature overwrites oidc.KeySet.VerifySignature
func (s *staticKeySet) VerifySignature(ctx context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt)
	if err != nil {
		return nil, err
	}
	return jws.Verify(s.keys[0])
}

func TestToken(t *testing.T) {

	appConfig := &utils.AppConfig{
		OidcClientId:  "fake-oidc-client-id",
		OidcIssuerUrl: "https://fake-oidc-provider",
	}
	test := assert.New(t)

	t.Log("Test oidc token authentication.")

	tcs := []struct {
		name           string
		expectedStatus int
		authHeader     string
		signingKeyFile string
	}{
		{
			name:           "valid token",
			authHeader:     jwt.BuildAuthHeader(appConfig, false, dummySigningKeyFile, signingKeyPrivate, jwt.Claim{}),
			expectedStatus: http.StatusOK,
			signingKeyFile: dummySigningKeyFile,
		},
		{
			name:           "no authorization header",
			authHeader:     "",
			expectedStatus: http.StatusBadRequest,
			signingKeyFile: dummySigningKeyFile,
		},
		{
			name:           "no bearer token",
			authHeader:     "test: test",
			expectedStatus: http.StatusBadRequest,
			signingKeyFile: dummySigningKeyFile,
		},
		{
			name:           "no signature",
			authHeader:     "Bearer " + noSignatureToken,
			expectedStatus: http.StatusForbidden,
			signingKeyFile: dummySigningKeyFile,
		},
		{
			name:           "invalid signature",
			authHeader:     jwt.BuildAuthHeader(appConfig, false, dummySigningKeyFile, signingKeyPrivate, jwt.Claim{}),
			expectedStatus: http.StatusForbidden,
			signingKeyFile: invalidDummySigningKeyFile,
		},
		{
			name:           "expired token",
			authHeader:     jwt.BuildAuthHeader(appConfig, true, dummySigningKeyFile, signingKeyPrivate, jwt.Claim{}),
			expectedStatus: http.StatusForbidden,
			signingKeyFile: dummySigningKeyFile,
		},
		{
			name:           "invalid aud",
			authHeader:     jwt.BuildAuthHeader(appConfig, false, dummySigningKeyFile, signingKeyPrivate, jwt.Claim{Key: "aud", Value: "test"}),
			expectedStatus: http.StatusForbidden,
			signingKeyFile: dummySigningKeyFile,
		},
	}

	e := echo.New()
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "test123")
	}

	for _, tc := range tcs {

		t.Logf("\tTest %s:\tWhen checking for http status code %d", tc.name, tc.expectedStatus)

		req := httptest.NewRequest(echo.GET, "http://localhost/api/v1/clusters", nil)
		if tc.authHeader != "" {
			req.Header.Set(echo.HeaderAuthorization, tc.authHeader)
		}

		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		m := monitoring.NewMetrics("cluster_registry_api_authz_test", nil, true)
		auth, err := NewAuthenticator(appConfig, m)
		pubKeys := []*jose.JSONWebKey{jwt.GetSigningKey(tc.signingKeyFile, signingKeyPublic)}

		if err != nil {
			t.Fatalf("Failed to initialize authenticator: %v", err)
		}
		auth.setVerifier(oidc.NewVerifier(
			appConfig.OidcIssuerUrl,
			&staticKeySet{keys: pubKeys},
			&oidc.Config{ClientID: appConfig.OidcClientId},
		))

		h := auth.VerifyToken()(handler)
		test.NoError(h(c))
		assert.Equal(t, tc.expectedStatus, c.Response().Status)
	}
}
