#!/bin/sh
set -e

# Verificar que el directorio /data existe
if [ ! -d "/data" ]; then
    echo "Error: El directorio /data no está montado"
    exit 1
fi

# Verificar que tenemos permisos de escritura
if [ ! -w "/data" ]; then
    echo "Error: No se tienen permisos de escritura en /data"
    exit 1
fi

echo "Iniciando la aplicación..."

# Ejecutar la aplicación con los argumentos proporcionados
exec "$@"
