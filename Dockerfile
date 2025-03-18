FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o ekz-tesla .

FROM scratch
COPY --from=builder /app/ekz-tesla /
ENTRYPOINT ["/ekz-tesla"]
