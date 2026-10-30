{{/* Имя чарта */}}
{{- define "kubron.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Полное имя релиза */}}
{{- define "kubron.fullname" -}}
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

{{- define "kubron.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Метки селектора: не меняются между обновлениями (иначе Deployment не обновить) */}}
{{- define "kubron.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubron.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "kubron.labels" -}}
helm.sh/chart: {{ include "kubron.chart" . }}
{{ include "kubron.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "kubron.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kubron.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* Образ: digest важнее тега, тег по умолчанию — appVersion */}}
{{- define "kubron.image" -}}
{{- if .Values.image.digest }}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}
{{- end }}

{{/* Имя Secret с ключом шифрования и строкой подключения */}}
{{- define "kubron.secretName" -}}
{{- if .Values.secrets.create }}
{{- printf "%s-secrets" (include "kubron.fullname" .) }}
{{- else }}
{{- required "secrets.existingSecret is required: create a Secret with the encryption key and the database URL (docs/deploy.md), or set secrets.create=true for development only" .Values.secrets.existingSecret }}
{{- end }}
{{- end }}

{{/* Доверять X-Forwarded-Proto: null — автоматически (ingress с TLS), иначе значение как есть */}}
{{- define "kubron.trustProxy" -}}
{{- if kindIs "invalid" .Values.kubron.trustProxy }}
{{- if and .Values.ingress.enabled .Values.ingress.tls }}1{{ else }}0{{ end }}
{{- else }}
{{- if .Values.kubron.trustProxy }}1{{ else }}0{{ end }}
{{- end }}
{{- end }}

{{/* Токен ServiceAccount монтируется только в режиме in-cluster */}}
{{- define "kubron.automountToken" -}}
{{- if .Values.inCluster.enabled }}true{{ else }}false{{ end }}
{{- end }}

{{/* Имя кластерных объектов (ClusterRole/Binding): с namespace, иначе одноимённые релизы конфликтуют */}}
{{- define "kubron.clusterRoleName" -}}
{{- printf "%s-%s" (include "kubron.fullname" .) .Release.Namespace | trunc 63 | trimSuffix "-" }}
{{- end }}
