package static

// TemplateConfig includes the template for the OVPN server config file.
const TemplateConfig = `
user nobody
group nogroup

status /tmp/openvpn.log
{{ if eq .Protocol "UDP" -}}
explicit-exit-notify 1
{{ end -}}

server 192.168.255.0 255.255.255.0
port 1194
proto {{ .Protocol | lower }}
dev tun0

cert {{ .Files.TLSServerCrt }}
key {{ .Files.TLSServerKey }}
ca {{ .Files.TLSCaCrt }}
dh {{ .Files.DHParams }}
tls-crypt {{ .Files.TLSAuth }}
crl-verify {{ .Files.CRL }}

auth {{ .Security.Hmac }}
cipher {{ .Security.Cipher }}

keepalive 10 60
key-direction 0
persist-key
persist-tun
verb 3

push "route 192.168.255.0 255.255.255.0"
{{ range .Routes -}}
push "route {{ .IP }} {{ .Mask }}"
{{ end -}}
{{ range .Nameservers -}}
push "dhcp-option DNS {{ . }}"
{{ end -}}
{{ if .RedirectAll -}}
push "redirect-gateway def1"
{{ end -}}
`
