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
	"io/ioutil"

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

var certificate *x509.Certificate

func main() {

	data, err := ioutil.ReadFile("../../test/testdata/dummycertificate.pem")
	if err != nil {
		panic("Failed to read file.")
	}
	block, _ := pem.Decode(data)
	cert, err := x509.ParseCertificate(block.Bytes)

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
	kid := hex.EncodeToString(thumbprint)

	jwks := []JWKS{
		JWKS{
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
		panic("Failed to marshal struct.")
	}

	fmt.Println(string(jwksKeysJson))
}
