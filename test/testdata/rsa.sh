!/usr/loca/env bash

# generate private key and self signed certificate
openssl genrsa -out dummyRsaPrivateKey.pem 1024

# generate self signed certificate
openssl req -new -x509 -sha256 -key dummyRsaPrivateKey.pem -days 3650 -out cert.pem  -subj "/CN=fake-oidc-provider"




# 'x5t': X.509 Certificate SHA-1 Thumbprint
echo $(openssl x509 -in dummyCertificate.pem -fingerprint -noout)

# 'n' modulus 
openssl x509 -in dummyCertificate.pem -noout -modulus
