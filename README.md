# Toolbox API 🛠️

> **API de herramientas para potenciar tus proyectos de IA y automatización**

## ✨ Características

- **Único Endpoint**: Todas las herramientas accesibles a través de una sola API
- **Fácil Integración**: Compatible con cualquier lenguaje de programación
- **Open Source**: Código abierto y gratuito para la comunidad
- **Próximamente**: Componentes React/JSX listos para usar

## 🚀 Comenzar

### Opción 1: Autohospedado

```bash
# Clonar el repositorio
git clone https://github.com/tu-usuario/toolbox-api.git
cd toolbox-api

# Iniciar con Docker
make rebuild
```

### Opción 2: SaaS (Recomendado)

¿No quieres lidiar con el auto-hospedaje? Prueba nuestra versión alojada:

🔜 Próximamente en [toolbox-api.com](https://toolbox-api.com)

## 🧩 Componentes Próximamente

- [ ] Componente React para búsqueda web
- [ ] Integración con OpenAI
- [ ] Herramientas de análisis de texto
- [ ] Widgets configurables

## 📚 Documentación

```bash
# Ejemplo de uso con cURL
curl -X POST https://api.toolbox.com/v1/agent \
  -H "Authorization: Bearer TU_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "web_search",
    "query": "últimas noticias de IA"
  }'
```

## 🤝 Contribuir

Las contribuciones son bienvenidas. Por favor, lee nuestras [guías de contribución](CONTRIBUTING.md) para más detalles.

## 📄 Licencia

MIT © [blissito](https://github.com/blissito)

---

💡 **Nota**: Esta es una versión autohospedable. Para una solución lista para producción sin configuración, visita pronto [toolbox-api.com](https://toolbox-api.com)
