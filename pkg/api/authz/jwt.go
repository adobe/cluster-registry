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
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"golang.org/x/net/context"
)

const (
	egressTarget = "azure_ad"
)

var (
	clientID    = os.Getenv("OIDC_CLIENT_ID")
	issuerURL   = os.Getenv("OIDC_ISSUER_URL")
	tokenLookup = "Authorization"
	authScheme  = "Bearer"
)

// Authenticator implements the OIDC authentication
type Authenticator struct {
	verifier *oidc.IDTokenVerifier
	ctx      context.Context
	met      monitoring.MetricsI
}

// NewAuthenticator creates new Authenticator
func NewAuthenticator(m monitoring.MetricsI) (*Authenticator, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, issuerURL)

	if err != nil {
		return nil, fmt.Errorf("init verifier failed: %v", err)
	}

	config := &oidc.Config{
		ClientID: clientID,
	}

	verifier := provider.Verifier(config)
	return &Authenticator{
		verifier: verifier,
		ctx:      ctx,
		met:      m,
	}, nil
}

// VerifyToken verifies if the JWT token from request header is valid
func (a *Authenticator) VerifyToken() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authorization := c.Request().Header.Get(tokenLookup)

			rawToken, err := extractToken(authorization)
			if err != nil {
				return c.JSON(http.StatusBadRequest, utils.NewError(err))
			}

			start := time.Now()
			token, err := a.verifier.Verify(a.ctx, rawToken)
			elapsed := float64(time.Since(start)) / float64(time.Second)

			a.met.RecordEgressRequestCnt(egressTarget)
			a.met.RecordEgressRequestDur(egressTarget, elapsed)

			if err != nil {
				return c.JSON(http.StatusForbidden, utils.NewError(err))
			}

			var claims struct {
				Oid string `json:"oid"`
			}
			if err := token.Claims(&claims); err != nil {
				return c.JSON(http.StatusForbidden, utils.NewError(err))
			}

			log.Info("Identity logged in: ", claims.Oid)
			return next(c)
		}
	}
}

// setVerifier set a custom verifier - used in testing
func (a *Authenticator) setVerifier(v *oidc.IDTokenVerifier) {
	a.verifier = v
}

// extractToken extracts the JWT token from authorization header data
func extractToken(authorization string) (string, error) {
	l := len(authScheme)
	if len(authorization) > l+1 && authorization[:l] == authScheme {
		return authorization[l+1:], nil
	}
	return "", errors.New("missing or malformed jwt")
}
