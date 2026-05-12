# Build
FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod ./
# go.sum se generará al primer `go mod download` real durante el build local.
# Si ya existe, copialo.
COPY go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/America/Lima /etc/localtime && \
    echo "America/Lima" > /etc/timezone
WORKDIR /app
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]
