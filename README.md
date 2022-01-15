# Cluster Registry API

[![tests](https://github.com/adobe/cluster-registry/actions/workflows/tests.yml/badge.svg)](https://github.com/adobe/cluster-registry/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/adobe/cluster-registry)](https://goreportcard.com/report/github.com/adobe/cluster-registry)
[![GoDoc](https://pkg.go.dev/badge/github.com/adobe/cluster-registry?status.svg)](https://pkg.go.dev/github.com/adobe/cluster-registry?tab=doc)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/adobe/cluster-registry)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/adobe/cluster-registry)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Cluster Registry is a Rest API representing the source of record for all Kubernetes clusters in your infrastructure fleet.
As the number of clusters grows, the need for automation to build, upgrade and operate those clusters also grows.
All clusters ar automatically registered and the information is accurately reflected in the Cluster Registry using a client-server architecture.

## Architecture diagram

![architecture](https://lucid.app/publicSegments/view/4b7b1961-92a4-484d-b9af-534fa1be3ba7/image.png)
