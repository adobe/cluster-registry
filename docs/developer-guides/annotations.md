# Annotations

You can add these Kubernetes annotations to cluster objects to customize Cluster Registry Client behavior.

Annotation keys and values can only be strings.

|Name                       | type |
|---------------------------|------|
|[clusters.registry.ethos.adobe.com/excluded-tags](#excluded-tags)|string|
|[clusters.registry.ethos.adobe.com/skip-ca-cert](#skip-ca-cert)|"true" or "false"|

## Excluded tags

The annotation `clusters.registry.ethos.adobe.com/excluded-tags` defines the behavior of the cluster-registry-client regarding to dynamic tags. If Alertmanager sends a signal for a specific tag, it will be ignored by Cluster Registry Client.

Example:
    `clusters.registry.ethos.adobe.com/excluded-tags: "onboarding,scaling"`

## Skip CA Cert

The annotation `clusters.registry.ethos.adobe.com/skip-ca-cert` defines the behavior of the cluster-registry-client for setting the K8s API CA Certificate. If it's set to `true`, the `CertificateAuthorityData` will not be set.

Example:
    `clusters.registry.ethos.adobe.com/skip-ca-cert: "true"`
