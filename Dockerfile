FROM --platform=$BUILDPLATFORM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o ekz-tesla-$TARGETARCH .

FROM alpine:latest AS certs
RUN apk --no-cache add ca-certificates

FROM scratch
ARG TARGETARCH
COPY --from=builder /app/ekz-tesla-$TARGETARCH /ekz-tesla
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/ekz-tesla"]
