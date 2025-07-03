#!/bin/sh
set -e

echo "Verificando permisos del directorio /data..."

# Solo verificar que el directorio exista
if [ ! -d "/data" ]; then
    echo "Error: El directorio /data no existe"
    exit 1
fi

# Verificar permisos de escritura en el directorio
if [ ! -w "/data" ]; then
    echo "Advertencia: No se tienen permisos de escritura en /data"
    # Continuamos de todos modos para ver si podemos escribir en archivos individuales
fi

# Verificar archivos de la base de datos
check_db_file() {
    local file="$1"
    if [ -f "$file" ]; then
        echo "Verificando permisos de $file"
        if [ ! -w "$file" ]; then
            echo "Advertencia: No se tienen permisos de escritura en $file"
            # Intentar cambiar el modo del archivo si es posible
            chmod +w "$file" 2>/dev/null || true
        fi
    else
        echo "El archivo $file no existe, se creará si es necesario"
        touch "$file" 2>/dev/null || echo "No se pudo crear $file"
    fi
}

# Verificar cada archivo de la base de datos
check_db_file "/data/toolbox.db"
check_db_file "/data/toolbox.db-wal"
check_db_file "/data/toolbox.db-shm"

echo "Estado actual del directorio /data:"
ls -la /data/

echo "Iniciando la aplicación como appuser..."

# Si el primer argumento es el comando a ejecutar, lo ejecutamos como appuser
if [ "$1" = "su" ] && [ "$2" = "-c" ]; then
    # Si ya estamos usando 'su -c', solo ejecutamos el comando
    shift 2
    exec "$@"
else
    # De lo contrario, ejecutamos como appuser
    exec su -c "$*" appuser
fi
