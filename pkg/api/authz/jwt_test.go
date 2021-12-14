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

package authz

import (
	"crypto"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	jose "gopkg.in/square/go-jose.v2"
)

const (
	rsaPrivateKeyType  = "RSA PRIVATE KEY"
	rsaPublicKeyType   = "RSA PUBLIC KEY"
	dummyRsaPrivateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBAPjSG8Csjiz3in2EAHOXd3Q6tbqmJiQ8hargzGdKJvOwevQzc7i+
FG0UDrGuRrM1xjgZHuCBHdQbBjq+WUsJsrECAwEAAQJAEyhqRp2CnOe6XAur1TqW
UfarQ2HDkgqu6AdC9bj54s1O68kktbN8/trbA4isTGquG5HQhV5JabprrXFhZNNS
oQIhAP/p6RYhtUnyaYzdrsC3TXYteDC2ddR7aKX0Av+ZHut9AiEA+OeV7tjgRTx5
gWcJ+6E/jBjWgANYKJJwjfFyQIVWQkUCIHRnf2BTwNR78Urj4wNB3XgtwofV1s7p
u3YRAfQlQA05AiEA6rFbA3qFhWMvYp+onxZ9F/l3j/8XSjJCZOTMCSBwpE0CIQDB
VE2n2slEYlMHln2TVBAInRCL62pwHNQc269J9Y9bSQ==
-----END RSA PRIVATE KEY-----
`
	invalidDummyRsaPrivateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBANGT0mwSY7+8gntZeUFexC2dpVOG81EBg/WLg1AnRTYTFTaYLFVO
TBxmH3zr/LiyKhcHcUBjhiob4lheJeg3k8sCAwEAAQJBAL2novVv0trRUdckSgmx
I6EQF2u2JPx6fZs4THW9g/GAwCWaT4cWKRzsjWdbHN5iFdLGaJqnu6Jx/Q/wSSh5
dtkCIQD/dtKLcalDIadFmeg0N1l9AQeKzqsZ8RMwwau04SuoNwIhANIEXAu3llc8
12/GK2dTQ23munaEOmr6lqUJWdt/+b8NAiAtWXiSzICRrD23e1TfQBwgtrgSChIR
rtwLQbYri/VmDQIhAK9lHq5Wc8OFt3LM8QDJA/5r/HvwcI1ZnKhWR+pOVfidAiEA
yVILFFMdumCAM2se/pV5rZq7e01UnH85py8Ba1Oe838=
-----END RSA PRIVATE KEY-----
`
	noSignatureToken = `
eyJhbGciOiJSUzI1NiIsImtpZCI6ImNkNTVlNTFiODM3YmMxM2Q4NzNjZmYxYTllY2ZmZTIyOTlkMTE1ZTAyOTUwYTM2ZTNiZDY2ZTVmZTBlNzNmNTYifQ.eyJhdWQiOiI2MDMyYjk4My1lZTVhLTQwOTYtOTk1Ny01NTczMmI5MmFiNDQiLCJleHAiOiIxNjE2MDg5NjU1IiwiaWF0IjoiMTYxNjA4NjA1NSIsImlwZCI6Imh0dHBzOi8vc3RzLndpbmRvd3MubmV0L2ZhN2IxYjVhLTdiMzQtNDM4Ny05NGFlLWQyYzE3OGRlY2VlMS8iLCJpc3MiOiJodHRwczovL3N0cy53aW5kb3dzLm5ldC9mYTdiMWI1YS03YjM0LTQzODctOTRhZS1kMmMxNzhkZWNlZTEvIiwib2lkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAwIn0
`
	dummyOid    = "00000000-0000-0000-0000-000000000000"
	expiredDate = "2021-03-11T00:00:00Z"
)

// getSigningKey converts rsaPrivateKey into a private/public JSONWebKey
func getSigningKey(rsaPrivateKey string, rsaKeyType string) *jose.JSONWebKey {
	var key *jose.JSONWebKey

	block, _ := pem.Decode([]byte(rsaPrivateKey))
	if block == nil {
		panic("failed to decode pem.")
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse the key: " + err.Error())
	}

	if rsaKeyType == rsaPrivateKeyType {
		key = &jose.JSONWebKey{Key: rsaKey, Use: "sig", Algorithm: string(jose.RS256)}
	} else {
		key = &jose.JSONWebKey{Key: rsaKey.Public(), Use: "sig", Algorithm: string(jose.RS256)}
	}

	thumbprint, err := key.Thumbprint(crypto.SHA256)
	if err != nil {
		panic("failed to compute thumbprint:" + err.Error())
	}

	key.KeyID = hex.EncodeToString(thumbprint)
	return key
}

type claim struct {
	key   string
	value string
}

// dummyToken represent the token claims
type dummyToken struct {
	claims map[string]string
}

// newDummyToken
func newDummyToken() *dummyToken {
	claims := make(map[string]string)
	claims["exp"] = fmt.Sprint(time.Now().Add(1 * time.Hour).Unix())
	claims["iat"] = fmt.Sprint(time.Now().Unix())
	claims["iss"] = issuerURL
	claims["ipd"] = issuerURL
	claims["aud"] = clientID
	claims["oid"] = dummyOid

	return &dummyToken{claims: claims}
}

// setExpiration sets the token expiration
func (t *dummyToken) setExpiration(tm time.Time) {
	t.claims["exp"] = fmt.Sprint(tm.Unix())
}

// setClaim sets a token claim
func (t *dummyToken) setClaim(c claim) {
	t.claims[c.key] = c.value
}

// signToken
func (t *dummyToken) sign() string {
	signingKey := getSigningKey(dummyRsaPrivateKey, rsaPrivateKeyType)
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       signingKey,
	}, nil)
	if err != nil {
		panic(err.Error())
	}

	claimString, err := json.Marshal(t.claims)
	if err != nil {
		panic(err.Error())
	}

	signedToken, err := signer.Sign([]byte(claimString))
	if err != nil {
		panic(err.Error())
	}

	serializedToken, err := signedToken.CompactSerialize()
	if err != nil {
		panic(err.Error())
	}
	return serializedToken
}

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

// buildAuthHeader builds the authorization header with a JWT bearer token
func buildAuthHeader(expiredToken bool, c claim) string {
	dt := newDummyToken()

	if expiredToken == true {
		expiration, _ := time.Parse(time.RFC3339Nano, expiredDate)
		dt.setExpiration(expiration)
	}

	if c.key != "" {
		dt.setClaim(c)
	}

	signedToken := dt.sign()
	return authScheme + " " + signedToken
}

func TestToken(t *testing.T) {

	test := assert.New(t)
	tcs := []struct {
		name          string
		code          int
		authHeader    string
		rsaSigningKey string
	}{
		{
			name:          "valid token",
			authHeader:    buildAuthHeader(false, claim{}),
			code:          http.StatusOK,
			rsaSigningKey: dummyRsaPrivateKey,
		},
		{
			name:          "no authorization header",
			authHeader:    "",
			code:          http.StatusBadRequest,
			rsaSigningKey: dummyRsaPrivateKey,
		},
		{
			name:          "no bearer token",
			authHeader:    "test: test",
			code:          http.StatusBadRequest,
			rsaSigningKey: dummyRsaPrivateKey,
		},
		{
			name:          "no signature",
			authHeader:    authScheme + " " + noSignatureToken,
			code:          http.StatusForbidden,
			rsaSigningKey: dummyRsaPrivateKey,
		},
		{
			name:          "invalid signature",
			authHeader:    buildAuthHeader(false, claim{}),
			code:          http.StatusForbidden,
			rsaSigningKey: invalidDummyRsaPrivateKey,
		},
		{
			name:          "expired token",
			authHeader:    buildAuthHeader(true, claim{}),
			code:          http.StatusForbidden,
			rsaSigningKey: dummyRsaPrivateKey,
		},
		{
			name:          "invalid aud",
			authHeader:    buildAuthHeader(false, claim{key: "aud", value: "test"}),
			code:          http.StatusForbidden,
			rsaSigningKey: dummyRsaPrivateKey,
		},
	}

	e := echo.New()
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "test123")
	}

	for _, tc := range tcs {
		req := httptest.NewRequest(echo.GET, "http://localhost/api/v1/clusters", nil)
		if tc.authHeader != "" {
			req.Header.Set(echo.HeaderAuthorization, tc.authHeader)
		}
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)

		m := monitoring.NewMetrics("cluster_registry_api_authz_test", nil, true)
		auth, err := NewAuthenticator(m)
		pubKeys := []*jose.JSONWebKey{getSigningKey(tc.rsaSigningKey, rsaPublicKeyType)}

		if err != nil {
			t.Fatalf("Failed to initialize authenticator: %v", err)
		}
		auth.setVerifier(oidc.NewVerifier(
			issuerURL,
			&staticKeySet{keys: pubKeys},
			&oidc.Config{ClientID: clientID},
		))

		h := auth.VerifyToken()(handler)
		test.NoError(h(c))
		assert.Equal(t, tc.code, c.Response().Status)
	}
}
