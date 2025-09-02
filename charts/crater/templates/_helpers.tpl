{{/*
Generate the url of grafana proxy
*/}}
{{- define "crater.grafanaProxyURL" -}}
{{- $protocol := .Values.protocol -}}
{{- $host := .Values.grafanaProxy.host -}}
{{- printf "%s://%s" $protocol $host -}}
{{- end -}}

{{/*
Process grafana URLs by concatenating baseURL with paths
*/}}
{{- define "crater.processGrafanaURLs" -}}
{{- $baseURL := include "crater.grafanaProxyURL" . -}}
{{- $grafanaConfig := .Values.frontendConfig.grafana -}}
{{- $result := dict -}}
{{- range $key, $value := $grafanaConfig -}}
  {{- if ne $key "baseURL" -}}
    {{- if kindIs "map" $value -}}
      {{- $nestedResult := dict -}}
      {{- range $nestedKey, $nestedValue := $value -}}
        {{- $_ := set $nestedResult $nestedKey (printf "%s%s" $baseURL $nestedValue) -}}
      {{- end -}}
      {{- $_ := set $result $key $nestedResult -}}
    {{- else -}}
      {{- $_ := set $result $key (printf "%s%s" $baseURL $value) -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- $result | toJson -}}
{{- end -}}

{{/*
Generate the complete frontend config with processed grafana URLs
*/}}
{{- define "crater.frontendConfig" -}}
{{- $config := deepCopy .Values.frontendConfig -}}
{{- $grafanaURLs := include "crater.processGrafanaURLs" . | fromJson -}}
{{- $_ := set $config "grafana" $grafanaURLs -}}
{{- $config | toJson -}}
{{- end -}}

{{/*
Generate the url of main project
*/}}
{{- define "crater.mainURL" -}}
{{- $protocol := .Values.protocol -}}
{{- $host := .Values.host -}}
{{- printf "%s://%s" $protocol $host -}}
{{- end -}}

{{/*
Generate storage server configuration from backendConfig
*/}}
{{- define "crater.storageServerConfig" -}}
postgres:
  {{ toYaml .Values.backendConfig.postgres | nindent 2 | trim }}
userSpacePrefix: {{ .Values.backendConfig.storage.prefix.user }}
accountSpacePrefix: {{ .Values.backendConfig.storage.prefix.account }}
publicSpacePrefix: {{ .Values.backendConfig.storage.prefix.public }}
auth:
  {{ toYaml .Values.backendConfig.auth | nindent 2 | trim }}
{{- end -}}

{{/*
Generate dockerconfigjson
*/}}
{{- define "dockerconfigjson" -}}
{{- $registry := .Values.backendConfig.registry.harbor.server -}}
{{- $username := .Values.backendConfig.registry.harbor.user -}}
{{- $password := .Values.backendConfig.registry.harbor.password -}}
{{- printf "{\"auths\":{\"%s\":{\"username\":\"%s\",\"password\":\"%s\",\"auth\":\"%s\"}}}" $registry $username $password (printf "%s:%s" $username $password | b64enc) | b64enc -}}
{{- end -}}

{{/*
Generate backend config with images from top-level images section
*/}}
{{- define "crater.backendConfig" -}}
{{- $config := deepCopy .Values.backendConfig -}}
{{- if $config.registry.enable -}}
  {{- $buildTools := $config.registry.buildTools -}}
  {{- $_ := set $buildTools "images" (dict 
    "buildx" (printf "%s:%s" .Values.images.buildx.repository .Values.images.buildx.tag)
    "nerdctl" (printf "%s:%s" .Values.images.nerdctl.repository .Values.images.nerdctl.tag)
    "envd" (printf "%s:%s" .Values.images.envd.repository .Values.images.envd.tag)
  ) -}}
  {{- $_ := set $config.registry "buildTools" $buildTools -}}
{{- end -}}
{{- $_ := set $config "host" .Values.host -}}
{{- $_ := set $config "namespaces" (dict "job" .Values.namespaces.job "image" .Values.namespaces.image) -}}
{{- $config | toYaml -}}
{{- end -}}


