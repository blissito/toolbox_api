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

# Copiar el código fuente
COPY . .

# Construir la aplicación
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

# Crear usuario y grupo no root
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Crear directorios necesarios con los permisos correctos
RUN mkdir -p /app/static/css /app/static/js /app/static/images /app/static/assets \
    /app/data \
    /app/docs \
    && chown -R appuser:appgroup /app \
    && chmod -R 755 /app

# Copiar archivos desde el builder
COPY --from=builder --chown=appuser:appgroup /app/toolbox-api /app/
COPY --from=builder --chown=appuser:appgroup /app/static/ /app/static/
COPY --from=builder --chown=appuser:appgroup /app/home.html /app/
COPY --from=builder --chown=appuser:appgroup /app/docs/ /app/docs/

# Asegurar que los directorios tengan los permisos correctos
RUN chmod -R 755 /app/static && \
    chmod 755 /app/home.html && \
    chmod -R 755 /app/docs && \
    chmod +x /app/toolbox-api

# Usar el usuario no root
USER appuser

# Puerto expuesto
EXPOSE 8000

# Variables de entorno
ENV TZ=UTC \
    PORT=8000

# Comando por defecto
CMD ["/app/toolbox-api"]
