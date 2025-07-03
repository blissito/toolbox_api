// Usar Preact desde el ámbito global
const { h, render } = window.preact || window;
const { useState, useEffect } = window.preactHooks || window;

// Componente principal del dashboard
function Dashboard() {
  const [isLoading, setIsLoading] = useState(true);
  const [message, setMessage] = useState('Cargando...');

  // Efecto para cargar datos iniciales
  useEffect(() => {
    // Simular carga de datos
    const timer = setTimeout(() => {
      setIsLoading(false);
      setMessage('Bienvenido al Dashboard de Toolbox API');
    }, 1000);

    return () => clearTimeout(timer);
  }, []);

  return h('div', { className: 'container mx-auto px-4 py-8' },
    // Encabezado
    h('header', { className: 'mb-8 text-center' },
      h('h1', { className: 'text-3xl font-bold mb-2' }, 'Toolbox API'),
      h('p', { className: 'text-gray-400' }, message)
    ),

    // Contenido principal
    h('main', { className: 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6' },
      // Tarjeta de Documentación
      h('div', { className: 'bg-card-bg rounded-lg p-6 shadow-lg' },
        h('h2', { className: 'text-xl font-semibold mb-3' }, 'Documentación'),
        h('p', { className: 'text-gray-400 mb-4' }, 'Consulta la documentación completa de la API para integrarla en tus proyectos.'),
        h('a', 
          { 
            href: '/docs', 
            className: 'inline-block bg-neon-blue text-black font-medium py-2 px-4 rounded hover:bg-opacity-90 transition-colors',
            'data-navigo': ''
          },
          'Ver Documentación'
        )
      ),

      // Tarjeta de WebFetch
      h('div', { className: 'bg-card-bg rounded-lg p-6 shadow-lg' },
        h('h2', { className: 'text-xl font-semibold mb-3' }, 'WebFetch'),
        h('p', { className: 'text-gray-400 mb-4' }, 'Documentación detallada del endpoint WebFetch para extraer contenido web.'),
        h('a', 
          { 
            href: '/docs/webfetch', 
            className: 'inline-block bg-neon-green text-black font-medium py-2 px-4 rounded hover:bg-opacity-90 transition-colors',
            'data-navigo': ''
          },
          'Ver WebFetch'
        )
      ),

      // Tarjeta de Estado del Servicio
      h('div', { className: 'bg-card-bg rounded-lg p-6 shadow-lg' },
        h('h2', { className: 'text-xl font-semibold mb-3' }, 'Estado del Servicio'),
        h('div', { className: 'flex items-center mb-4' },
          h('span', { className: 'w-3 h-3 bg-green-500 rounded-full mr-2' }),
          h('span', { className: 'text-green-400' }, 'Operativo')
        ),
        h('p', { className: 'text-gray-400' }, 'Todos los sistemas funcionando correctamente.')
      )
    ),

    // Pie de página
    h('footer', { className: 'mt-12 text-center text-gray-500 text-sm' },
      '© 2025 Toolbox API - Todos los derechos reservados'
    )
  );
}

// Renderizar el dashboard cuando el DOM esté listo
document.addEventListener('DOMContentLoaded', () => {
  const appContainer = document.getElementById('app');
  if (appContainer) {
    render(h(Dashboard), appContainer);
  }
});
