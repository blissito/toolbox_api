# Etapa de construcción
FROM golang:1.23-bullseye AS builder

WORKDIR /app

# Instalar dependencias básicas
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Configurar Go para compilación estática sin CGO
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct

# Copiar archivos de dependencias primero para caché
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download \
    && go mod verify

# Copiar el código fuente y construir
COPY . .
RUN go build -v -ldflags='-w -s' -o /app/toolbox-api .

# Etapa final
FROM debian:bullseye-slim

WORKDIR /app

# Instalar solo lo esencial para ejecución
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Crear directorio para datos
RUN mkdir -p /data

# Copiar archivos desde el builder
COPY --from=builder /app/toolbox-api /app/
COPY --from=builder /app/static /app/static
COPY --from=builder /app/home.html /app/
COPY --from=builder /app/docs /app/docs
COPY --from=builder /app/init-db.sh /app/

# Hacer ejecutables los archivos necesarios
RUN chmod +x /app/toolbox-api /app/init-db.sh

# Puerto expuesto
EXPOSE 8000

# Variables de entorno
ENV TZ=UTC \
    PORT=8000

# Punto de entrada
ENTRYPOINT ["/app/init-db.sh"]

# Comando por defecto
CMD ["/app/toolbox-api"]
