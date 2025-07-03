#!/bin/bash
set -e

# Variables
APP_NAME="toolbox-api"
REGION="dfw"  # Puedes cambiarlo a tu región preferida
VOLUME_NAME="toolbox_data"
VOLUME_SIZE=1  # en GB

# Verificar si la aplicación ya existe
if ! flyctl status --app $APP_NAME &>/dev/null; then
    echo "Creando aplicación $APP_NAME..."
    flyctl launch --name $APP_NAME --region $REGION --no-deploy --dockerfile Dockerfile
    
    # Crear volumen
    echo "Creando volumen de $VOLUME_SIZE GB..."
    flyctl volumes create $VOLUME_NAME \
        --app $APP_NAME \
        --region $REGION \
        --size $VOLUME_SIZE \
        --no-encryption  # Desactivar encriptación para mejor rendimiento
    
    # Configurar variables de entorno
    echo "Configurando variables de entorno..."
    flyctl secrets set \
        --app $APP_NAME \
        JWT_SECRET=$(openssl rand -hex 32) \
        FLY=true
    
    echo "¡Aplicación creada! Desplegando..."
    flyctl deploy --app $APP_NAME
else
    echo "La aplicación $APP_NAME ya existe. Actualizando..."
    flyctl deploy --app $APP_NAME
fi

echo "¡Despliegue completado!"
echo "Puedes abrir la aplicación con: flyctl open --app $APP_NAME"
