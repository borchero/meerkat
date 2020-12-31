FROM alpine:3.12

RUN apk add --no-cache openvpn=2.4.9-r0
ENTRYPOINT ["/app/entrypoint.sh"]
