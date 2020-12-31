package static

// TemplateClient contains the template for the OVPN client file.
const TemplateClient = `
client
nobind
dev tun
remote-cert-tls server
remote {{ .Host }} {{ .Port }} {{ .Protocol | lower }}

<key>
{{ .Secrets.TLSClientKey | trim }}
</key>
<cert>
{{ .Secrets.TLSClientCrt | trim }}
</cert>
<ca>
{{ .Secrets.TLSCaCrt | trim }}
</ca>

<tls-crypt>
{{ .Secrets.TLSAuth | trim }}
</tls-crypt>

auth {{ .Security.Hmac }}
cipher {{ .Security.Cipher }}
`
