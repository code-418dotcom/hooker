# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /hooker ./cmd/hooker

# ---- runtime stage ----
FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /hooker /hooker
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER nobody

LABEL hooker.protect="true"

ENTRYPOINT ["/hooker"]
