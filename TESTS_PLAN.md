# Plan de Pruebas - API Toolbox

Este documento detalla el plan de pruebas para la API de Toolbox, enfocándose en los endpoints y funcionalidades principales que utilizarán los clientes.

## 1. Tests de Autenticación

### 1.1 Test de solicitud de Magic Link
- **Endpoint**: `POST /api/auth/request-magic-link`
  - ✓ Debe aceptar un email válido y devolver un mensaje de éxito
  - ✓ Debe rechazar emails inválidos
  - ✓ Debe rechazar peticiones sin email
  - ✓ Debe rechazar métodos diferentes a POST

### 1.2 Test de validación de Magic Link
- **Endpoint**: `GET /api/auth/validate`
  - ✓ Debe validar correctamente un token válido
  - ✓ Debe rechazar tokens inválidos
  - ✓ Debe rechazar tokens expirados
  - ✓ Debe establecer correctamente las cookies de sesión

## 2. Tests de Gestión de API Keys

### 2.1 Test de creación de API Key
- **Endpoint**: `POST /api/keys/create`
  - ✓ Debe crear una nueva API key con nombre válido
  - ✓ Debe requerir autenticación
  - ✓ Debe rechazar nombres vacíos
  - ✓ Debe devolver el formato correcto de la API key

### 2.2 Test de listado de API Keys
- **Endpoint**: `GET /api/keys/list`
  - ✓ Debe listar todas las API keys del usuario
  - ✓ Debe requerir autenticación
  - ✓ Debe mostrar correctamente el estado de cada key (activa/revocada)
  - ✓ Debe incluir la fecha de último uso

### 2.3 Test de revocación de API Key
- **Endpoint**: `POST /api/keys/revoke`
  - ✓ Debe revocar correctamente una API key existente
  - ✓ Debe requerir autenticación
  - ✓ Debe rechazar la revocación de keys inexistentes
  - ✓ Debe rechazar la revocación de keys de otros usuarios

## 3. Tests de Herramienta WebFetch

### 3.1 Test de WebFetch
- **Endpoint**: `POST /api/tool` (tool: "webfetch")
  - ✓ Debe obtener contenido HTML correctamente
  - ✓ Debe manejar diferentes formatos de salida (html, text, markdown)
  - ✓ Debe extraer metadatos correctamente (título, descripción, imagen)
  - ✓ Debe manejar timeouts apropiadamente
  - ✓ Debe validar URLs incorrectas
  - ✓ Debe requerir autenticación (API key o sesión)
  - ✓ Debe respetar límites de tiempo máximo

## 4. Tests de Seguridad

### 4.1 Test de Autenticación
- ✓ Debe validar correctamente tokens JWT
- ✓ Debe validar correctamente API keys
- ✓ Debe rechazar tokens expirados
- ✓ Debe rechazar API keys revocadas

### 4.2 Test de Rate Limiting
- ✓ Debe aplicar límites de rata apropiados
- ✓ Debe manejar correctamente múltiples peticiones simultáneas

## 5. Tests de Manejo de Errores

### 5.1 Test de Respuestas de Error
- ✓ Debe devolver códigos HTTP apropiados
- ✓ Debe incluir mensajes de error descriptivos
- ✓ Debe mantener consistencia en el formato de respuesta de error
- ✓ Debe incluir códigos de error específicos para cada situación

## Ejemplos de Implementación

### Ejemplo de test con curl para creación de API Key
```shell
curl -X POST \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Key"}' \
  http://localhost:8080/api/keys/create
```

### Ejemplo de test con curl para WebFetch
```shell
curl -X POST \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "webfetch",
    "payload": {
      "url": "https://example.com",
      "format": "markdown",
      "timeout": 30
    }
  }' \
  http://localhost:8080/api/tool
```

## Recomendaciones de Implementación

1. Usar un framework de testing como `go test` para Go
2. Crear un ambiente de pruebas separado
3. Mockear servicios externos (especialmente para webfetch)
4. Implementar fixtures para datos de prueba
5. Usar variables de entorno para configuraciones
6. Documentar los casos de prueba y resultados esperados
7. Incluir pruebas de integración además de unitarias
8. Implementar cobertura de código para asegurar testing completo

## Notas Adicionales

- Todos los tests deben ser reproducibles y automatizables
- Se debe mantener un ambiente de pruebas limpio entre ejecuciones
- Los tests deben ser independientes entre sí
- Se debe documentar cualquier dependencia externa necesaria
- Se recomienda implementar tests de integración continua (CI)