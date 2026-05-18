{{- define "edge-dispatch.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edge-dispatch.fullname" -}}
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

{{- define "edge-dispatch.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edge-dispatch.labels" -}}
helm.sh/chart: {{ include "edge-dispatch.chart" . }}
{{ include "edge-dispatch.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "edge-dispatch.selectorLabels" -}}
app.kubernetes.io/name: {{ include "edge-dispatch.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "edge-dispatch.controlPlaneUrl" -}}
http://{{ .Release.Name }}-control-plane:{{ .Values.controlPlane.service.port }}
{{- end }}

{{- define "edge-dispatch.originUrl" -}}
http://{{ .Release.Name }}-origin:{{ .Values.origin.service.port }}
{{- end }}
