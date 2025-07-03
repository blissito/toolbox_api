#!/bin/bash

# Detener y eliminar el contenedor si ya existe
podman stop toolbox-api 2>/dev/null
podman rm toolbox-api 2>/dev/null

# Construir la imagen (opcional, descomenta si necesitas reconstruir)
# podman build -t toolbox-api .

# Ejecutar el contenedor con las variables de entorno
echo "Iniciando el contenedor..."
podman run --env-file=.env -p 8000:8000 --name toolbox-api -d toolbox-api

echo "\nServidor iniciado en http://localhost:8000"
echo "\nPara ver los logs del contenedor:"
echo "podman logs -f toolbox-api"
echo "\nPara detener el contenedor:"
echo "podman stop toolbox-api"
echo "\nPara iniciar el contenedor de nuevo:"
echo "podman start toolbox-api"
