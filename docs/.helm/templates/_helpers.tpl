{{- define "resources" }}
resources:
  requests:
    memory: {{ pluck .Values.werf.env .Values.resources.requests.memory | first | default .Values.resources.requests.memory._default }}
  limits:
    memory: {{ pluck .Values.werf.env .Values.resources.requests.memory | first | default .Values.resources.requests.memory._default }}
{{- end }}

{{- define "targetCluster" -}}
{{- $targetCluster := .Values.global.targetCluster | default "eu" -}}
{{- if and (eq .Values.global.env "production") (not (has $targetCluster (list "eu" "ru"))) -}}
{{- fail (printf "unsupported global.targetCluster %q: expected ru or eu" $targetCluster) -}}
{{- end -}}
{{- $targetCluster -}}
{{- end }}

{{- define "ruHost" -}}
{{- printf "ru.%s" . -}}
{{- end }}
