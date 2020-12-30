FROM golang:1.15 as builder

WORKDIR /app
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on

COPY go.mod go.mod
COPY hack.go hack.go
COPY cmd/main.go cmd/main.go
COPY pkg pkg

RUN go build -a -o operator cmd/main.go

#--------------------------------------------------------------------------------------------------

FROM alpine:3.12

RUN apk add --no-cache openvpn=2.4.9-r0 openssl=1.1.1i-r0
COPY --from=builder /app/operator /operator

ENTRYPOINT ["/operator"]
