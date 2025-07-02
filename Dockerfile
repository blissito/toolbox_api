# Etapa de construcción
FROM golang:1.24-alpine AS builder

# Establecer directorio de trabajo
WORKDIR /app

# Copiar solo los archivos necesarios para las dependencias
COPY go.mod .

# Descargar dependencias
RUN go mod download

# Copiar el resto de los archivos
COPY . .

# Construir la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -o toolbox-api .

# Etapa final
FROM alpine:3.18
WORKDIR /app

# Instalar dependencias necesarias
RUN apk --no-cache add ca-certificates

# Copiar el binario
COPY --from=builder /app/toolbox-api .

# Copiar archivos estáticos
COPY --from=builder /app/index.html .
COPY --from=builder /app/dashboard.html .
COPY --from=builder /app/docs ./docs/

# Puerto expuesto
EXPOSE 8000

# Comando para ejecutar la aplicación
CMD ["./toolbox-api"]
