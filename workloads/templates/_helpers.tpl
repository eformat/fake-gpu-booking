{{- define "gpu-test-workloads.labels" -}}
kueue.x-k8s.io/queue-name: {{ if .Values.reserved }}reserved{{ else }}unreserved{{ end }}
opendatahub.io/dashboard: "true"
{{- with .Values.extraLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "gpu-test-workloads.annotations" -}}
opendatahub.io/hardware-profile-name: {{ if .Values.reserved }}reserved-gpu{{ else }}unreserved-gpu{{ end }}
opendatahub.io/hardware-profile-namespace: {{ default .Release.Namespace .Values.hardwareProfile.namespace }}
openshift.io/required-scc: {{ .Values.requiredSCC }}
{{- with .Values.extraAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "gpu-test-workloads.podSpec" -}}
containers:
  - name: gpu-container
    image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
    {{- with .Values.command }}
    command:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    env:
      - name: NODE_NAME
        valueFrom:
          fieldRef:
            fieldPath: spec.nodeName
    {{- with .Values.extraEnv }}
      {{- toYaml . | nindent 6 }}
    {{- end }}
    resources:
      limits:
        {{ .Values.gpuResourceType }}: "{{ .Values.gpuCount }}"
      requests:
        {{ .Values.gpuResourceType }}: "{{ .Values.gpuCount }}"
{{- with .Values.nodeSelector }}
nodeSelector:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.tolerations }}
tolerations:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}
