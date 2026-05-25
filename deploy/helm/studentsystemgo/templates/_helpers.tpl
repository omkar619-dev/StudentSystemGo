{{/*
Generate the app name. Defaults to chart name but can be overridden via .Values.nameOverride.
*/}}
{{- define "studentsystemgo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Generate the full resource name (release name + chart name).
Used as the `name:` on Deployments, Services, etc.
Truncated to 63 chars (k8s label limit).
*/}}
{{- define "studentsystemgo.fullname" -}}
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
Standard labels to attach to every resource.
Useful for `kubectl get pods -l app.kubernetes.io/name=studentsystemgo`.
*/}}
{{- define "studentsystemgo.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "studentsystemgo.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels — the SUBSET used to match Pods to their Deployment/Service.
These MUST be stable across upgrades — never change them.
*/}}
{{- define "studentsystemgo.selectorLabels" -}}
app.kubernetes.io/name: {{ include "studentsystemgo.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}