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

package main

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"gopkg.in/square/go-jose.v2"
)

type JWKS struct {
	Use     string   `json:"use,omitempty"`
	Kty     string   `json:"kty,omitempty"`
	Kid     string   `json:"kid,omitempty"`
	Alg     string   `json:"alg,omitempty"`
	N       string   `json:"n,omitempty"`
	E       string   `json:"e,omitempty"`
	X5c     []string `json:"x5c,omitempty"`
	X5tSHA1 string   `json:"x5t,omitempty"`
}

type Keys struct {
	Keys []JWKS `json:"keys,omitempty"`
}

func main() {

	data, err := os.ReadFile("../../test/testdata/dummycertificate.pem")

	if err != nil {
		log.Panicf("Failed to read file: %v", err.Error())
	}

	block, _ := pem.Decode(data)
	cert, err := x509.ParseCertificate(block.Bytes)

	if err != nil {
		log.Panicf("Failed to parse certificate: %v", err.Error())
	}

	x5c := []string{base64.StdEncoding.EncodeToString(cert.Raw)}

	fingerprint := sha1.Sum(cert.Raw)
	x5t := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%X", fingerprint)))
	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)

	n := base64.RawURLEncoding.EncodeToString(rsaPublicKey.N.Bytes())

	eb := make([]byte, 8)
	binary.BigEndian.PutUint64(eb, uint64(rsaPublicKey.E))
	bytes.TrimLeft(eb, "\x00")
	e := base64.RawURLEncoding.EncodeToString(eb)

	jwKey := jose.JSONWebKey{Key: rsaPublicKey, Use: "sig", Algorithm: string(jose.RS256)}
	thumbprint, err := jwKey.Thumbprint(crypto.SHA256)

	if err != nil {
		log.Panicf("Failed to generate thumbprint: %v", err.Error())
	}

	kid := hex.EncodeToString(thumbprint)

	jwks := []JWKS{
		{
			Alg:     "RS256",
			Use:     "sig",
			Kty:     "RSA",
			N:       n,
			E:       e,
			Kid:     kid,
			X5tSHA1: x5t,
			X5c:     x5c,
		},
	}

	jwksKeys := &Keys{
		Keys: jwks,
	}

	jwksKeysJson, err := json.Marshal(&jwksKeys)
	if err != nil {
		log.Panicf("Failed to marshal struct: %v", err.Error())
	}

	fmt.Println(string(jwksKeysJson))
}
