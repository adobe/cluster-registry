{{/*
Expand the name of the chart.
*/}}
{{- define "cluster-registry-client.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cluster-registry-client.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cluster-registry-client.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cluster-registry-client.labels" -}}
helm.sh/chart: {{ include "cluster-registry-client.chart" . }}
{{ include "cluster-registry-client.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{ include "cluster-registry-client.appLabels" . }}
{{ include "cluster-registry-client.componentLabels" . }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cluster-registry-client.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cluster-registry-client.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Cluster Registry client application label
*/}}
{{- define "cluster-registry-client.appLabels" -}}
app: cluster-registry-client
{{- end }}

{{/*
Cluster Registry component label
*/}}
{{- define "cluster-registry-client.componentLabels" -}}
component: cluster-registry
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "cluster-registry-client.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cluster-registry-client.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
