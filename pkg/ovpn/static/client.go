package static

// TemplateClient contains the template for the OVPN client file.
const TemplateClient = `
client
nobind
dev tun
remote-cert-tls server
remote {{ .Host }} {{ .Port }} {{ .Protocol | lower }}

<key>
{{ .Secrets.TLSClientKey }}
</key>
<cert>
{{ .Secrets.TLSClientCrt }}
</cert>
<ca>
{{ .Secrets.TLSCaCrt }}
</ca>

<tls-crypt>
{{ .Secrets.TLSAuth }}
</tls-crypt>

auth {{ .Security.Hmac }}
cipher {{ .Security.Cipher }}
`
