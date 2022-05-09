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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"golang.org/x/net/context"
)

const (
	egressTarget = "azure_ad"
)

var (
	tokenLookup = "Authorization"
	authScheme  = "Bearer"
)

// Authenticator implements the OIDC authentication
// We have two verifiers to allow tokens with or without the 'spn' token.
type Authenticator struct {
	verifier    *oidc.IDTokenVerifier
	spnVerifier *oidc.IDTokenVerifier
	ctx         context.Context
	metrics     monitoring.MetricsI
}

// NewAuthenticator creates new Authenticator
func NewAuthenticator(appConfig *config.AppConfig, m monitoring.MetricsI) (*Authenticator, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, appConfig.OidcIssuerUrl)

	if err != nil {
		return nil, fmt.Errorf("init verifier failed: %v", err)
	}

	config := &oidc.Config{
		ClientID: appConfig.OidcClientId,
	}

	spnConfig := &oidc.Config{
		ClientID: "spn:" + appConfig.OidcClientId,
	}

	verifier := provider.Verifier(config)
	spnVerifier := provider.Verifier(spnConfig)

	return &Authenticator{
		verifier:    verifier,
		spnVerifier: spnVerifier,
		ctx:         ctx,
		metrics:     m,
	}, nil
}

// verify check if the token is valid by both verifiers, to allow tokens with or without the 'spn' prefix
func (a *Authenticator) verify(ctx context.Context, rawToken string) (*oidc.IDToken, error) {
	token, err := a.verifier.Verify(ctx, rawToken)
	if err != nil && strings.Contains(err.Error(), "spn:") {
		token, err = a.spnVerifier.Verify(ctx, rawToken)
	}
	return token, err
}

// VerifyToken verifies if the JWT token from request header is valid
func (a *Authenticator) VerifyToken() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authorization := c.Request().Header.Get(tokenLookup)

			rawToken, err := extractToken(authorization)
			if err != nil {
				return c.JSON(http.StatusBadRequest, NewError(err))
			}

			start := time.Now()
			token, err := a.verify(a.ctx, rawToken)
			elapsed := float64(time.Since(start)) / float64(time.Second)

			a.metrics.RecordEgressRequestCnt(egressTarget)
			a.metrics.RecordEgressRequestDur(egressTarget, elapsed)

			if err != nil {
				return c.JSON(http.StatusForbidden, NewError(err))
			}

			var claims struct {
				Oid string `json:"oid"`
			}
			if err := token.Claims(&claims); err != nil {
				return c.JSON(http.StatusForbidden, NewError(err))
			}

			c.Set("oid", claims.Oid)

			log.Info("Identity logged in: ", claims.Oid)
			return next(c)
		}
	}
}

// setVerifier set a custom verifier - used in testing
func (a *Authenticator) setVerifiers(v *oidc.IDTokenVerifier, spnv *oidc.IDTokenVerifier) {
	a.verifier = v
	a.spnVerifier = spnv
}

// extractToken extracts the JWT token from authorization header data
func extractToken(authorization string) (string, error) {
	l := len(authScheme)
	if len(authorization) > l+1 && authorization[:l] == authScheme {
		return authorization[l+1:], nil
	}
	return "", errors.New("missing or malformed jwt")
}

// Error struct
type Error struct {
	Errors map[string]interface{} `json:"errors"`
}

// NewError constructor
func NewError(err error) Error {
	e := Error{}
	e.Errors = make(map[string]interface{})
	switch v := err.(type) {
	case *echo.HTTPError:
		e.Errors["body"] = v.Message
	default:
		e.Errors["body"] = v.Error()
	}
	return e
}
