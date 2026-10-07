# Etapa 1: compilación
FROM golang:1.22-slim AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/microservicio-io .

# Etapa 2: runtime mínimo
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /out/microservicio-io /app/microservicio-io
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/app/microservicio-io"]
