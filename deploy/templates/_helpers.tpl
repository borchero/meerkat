{{- define "serviceAccount.name" -}}
{{- if .Values.rbac.serviceAccountName -}}
{{ .Values.rbac.serviceAccountName }}
{{- else -}}
{{ .Release.Name }}
{{- end -}}
{{- end -}}

{{- define "vault.agent.args" -}}
echo ${VAULT_CONFIG?} > /home/vault/config.json && vault agent -config=/home/vault/config.json
{{- end -}}

{{- define "vault.agent.config" -}}
{
  "auto_auth": {
    "method": {
      "type": "{{ .Values.vault.auth.type }}",
      "mount_path": "{{ .Values.vault.auth.mountPath }}",
      "config": {{ toJson .Values.vault.auth.config }}
    },
    "sink": [
      {
        "type": "file",
        "config": {
          "path": "/home/vault/.vault-token"
        }
      }
    ]
  },
  "exit_after_auth": false,
  "pid_file": "/home/vault/.pid",
  "vault": {
    {{- if .Values.vault.caCrt -}}
    "ca_cert": "{{ .Values.vault.caCrt }}",
    {{- end -}}
    "address": "{{ .Values.vault.address }}"
  },
  "template": [
    {
      "destination": "/vault/secrets/token",
      "contents": "{{ "{{" }} with secret \"auth/token/lookup-self\" {{ "}}" }}{{ "{{" }} .Data.id {{ "}}" }}\n{{ "{{" }} end {{ "}}" }}",
      "left_delimiter": "{{ "{{" }}",
      "right_delimiter": "{{ "}}" }}"
    }
  ]
}
{{- end -}}
