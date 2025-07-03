// Usar Preact desde el ámbito global
const { h, render } = window.preact || window;
const { useState, useEffect } = window.preactHooks || window;

// Componente de tarjeta de API Key
function ApiKeyCard({ apiKey, onCopy, onRevoke }) {
  return h('div', { class: 'bg-white rounded-lg shadow p-4 mb-4 flex justify-between items-center' },
    h('div', { class: 'flex-1' },
      h('div', { class: 'font-medium text-gray-900' }, 'Clave API'),
      h('div', { class: 'text-sm text-gray-500 font-mono bg-gray-50 p-2 rounded mt-1 truncate' }, apiKey)
    ),
    h('div', { class: 'flex space-x-2' },
      h('button', {
        onClick: onCopy,
        class: 'px-3 py-1 bg-blue-500 text-white text-sm rounded hover:bg-blue-600 transition-colors'
      }, 'Copiar'),
      h('button', {
        onClick: onRevoke,
        class: 'px-3 py-1 bg-red-500 text-white text-sm rounded hover:bg-red-600 transition-colors'
      }, 'Revocar')
    )
  );
}

// Componente de modal para crear nueva API Key
function CreateKeyModal({ isOpen, onClose, onCreate }) {
  const [keyName, setKeyName] = useState('');
  
  // Permisos fijos con todos los permisos habilitados
  const permissions = {
    read: true,
    write: true,
    delete: true
  };

  if (!isOpen) return null;

  const handleSubmit = (e) => {
    e.preventDefault();
    onCreate({
      name: keyName,
      permissions
    });
  };

  return h('div', { class: 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50' },
    h('div', { class: 'bg-white rounded-lg shadow-xl w-full max-w-md' },
      h('div', { class: 'p-6' },
        h('h3', { class: 'text-lg font-medium text-gray-900 mb-4' }, 'Crear nueva clave API'),
        h('form', { onSubmit: handleSubmit },
          h('div', { class: 'mb-4' },
            h('label', { class: 'block text-sm font-medium text-gray-700 mb-1' }, 'Nombre de la clave'),
            h('input', {
              type: 'text',
              value: keyName,
              onInput: (e) => setKeyName(e.target.value),
              class: 'w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500',
              placeholder: 'Ej: Aplicación Móvil',
              required: true
            })
          ),
          h('div', { class: 'mb-6' },
            h('p', { class: 'text-sm text-gray-500' }, 'Esta clave tendrá todos los permisos habilitados.')
          ),
          h('div', { class: 'flex justify-end space-x-3' },
            h('button', {
              type: 'button',
              onClick: onClose,
              class: 'px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
            }, 'Cancelar'),
            h('button', {
              type: 'submit',
              class: 'px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
            }, 'Crear clave')
          )
        )
      )
    )
  );
}

// Componente principal de la aplicación
function Dashboard() {
  const [isLoading, setIsLoading] = useState(true);
  const [apiKeys, setApiKeys] = useState([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newKey, setNewKey] = useState(null);

  // Cargar claves API al iniciar
  useEffect(() => {
    const loadApiKeys = async () => {
      try {
        const response = await fetch('/api/keys/list');
        
        if (response.redirected) {
          // Si el backend redirige, es porque requiere autenticación
          window.location.href = response.url;
          return;
        }
        
        if (response.ok) {
          const data = await response.json();
          setApiKeys(data.api_keys || []);
        } else {
          console.error('Error al cargar claves API');
        }
      } catch (error) {
        console.error('Error al cargar claves API:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadApiKeys();
  }, []);

  const handleCreateKey = async (keyData) => {
    try {
      const response = await fetch('/api/keys/create', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: keyData.name,
          permissions: Object.entries(keyData.permissions)
            .filter((_, value) => value)
            .map(([key]) => key)
        })
      });

      if (response.redirected) {
        // Si el backend redirige, es porque requiere autenticación
        window.location.href = response.url;
        return;
      }

      if (response.ok) {
        const data = await response.json();
        const newKey = {
          id: data.id || Date.now().toString(),
          key: data.api_key,
          name: keyData.name,
          permissions: Object.entries(keyData.permissions)
            .filter(([_, value]) => value)
            .map(([key]) => key),
          createdAt: new Date().toISOString().split('T')[0]
        };
        
        setApiKeys(prevKeys => [newKey, ...prevKeys]);
        setNewKey(newKey);
        setShowCreateModal(false);
      } else {
        const error = await response.json();
        alert(`Error al crear la clave: ${error.error || 'Error desconocido'}`);
      }
    } catch (error) {
      console.error('Error al crear la clave API:', error);
      alert('Error al conectar con el servidor');
    }
  };

  const handleCopyKey = (key) => {
    navigator.clipboard.writeText(key).then(() => {
      // Mostrar notificación de éxito
      const notification = document.createElement('div');
      notification.className = 'fixed bottom-4 right-4 bg-green-500 text-white px-4 py-2 rounded-md shadow-lg';
      notification.textContent = '¡Clave copiada al portapapeles!';
      document.body.appendChild(notification);
      
      // Eliminar la notificación después de 3 segundos
      setTimeout(() => {
        notification.remove();
      }, 3000);
    }).catch(err => {
      console.error('Error al copiar al portapapeles:', err);
      alert('No se pudo copiar la clave. Por favor, copia manualmente.');
    });
  };

  const handleCreateClick = async () => {
    // Verificar si el usuario está autenticado antes de mostrar el modal
    try {
      const response = await fetch('/api/auth/me', {
        headers: {
          'X-Requested-With': 'XMLHttpRequest'
        }
      });
      
      if (response.status === 200) {
        // Usuario autenticado, mostrar el modal
        setShowCreateModal(true);
      } else {
        // Si no está autenticado, redirigir al login
        window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname);
      }
    } catch (error) {
      console.error('Error al verificar autenticación:', error);
      window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname);
    }
  };

  const handleRevokeKey = async (keyId) => {
    if (!confirm('¿Estás seguro de que deseas revocar esta clave? Esta acción no se puede deshacer.')) {
      return;
    }

    try {
      const response = await fetch(`/api/keys/revoke/${keyId}`, {
        method: 'POST'
      });

      if (response.redirected) {
        // Si el backend redirige, es porque requiere autenticación
        window.location.href = response.url;
        return;
      }

      if (response.ok) {
        setApiKeys(prevKeys => prevKeys.filter(key => key.id !== keyId));
      } else {
        const error = await response.json();
        alert(`Error al revocar la clave: ${error.error || 'Error desconocido'}`);
      }
    } catch (error) {
      console.error('Error al revocar la clave API:', error);
      alert('Error al conectar con el servidor');
    }
  };

  if (isLoading) {
    return h('div', { class: 'min-h-screen flex items-center justify-center' },
      h('div', { class: 'animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500' })
    );
  }

  return h('div', { class: 'min-h-screen bg-gray-50' },
    // Header
    h('header', { class: 'bg-white shadow' },
      h('div', { class: 'max-w-7xl mx-auto px-4 py-4 sm:px-6 lg:px-8 flex justify-between items-center' },
        h('h1', { class: 'text-2xl font-bold text-gray-900' }, 'Dashboard de API Keys'),
        h('button', {
          onClick: handleCreateClick,
          class: 'px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
        }, 'Nueva clave API')
      )
    ),

    // Contenido principal
    h('main', { class: 'max-w-7xl mx-auto px-4 py-6 sm:px-6 lg:px-8' },
      // Sección de nueva clave generada (si existe)
      newKey && h('div', { class: 'mb-6 p-4 bg-green-50 border border-green-200 rounded-md' },
        h('h3', { class: 'text-sm font-medium text-green-800' }, '¡Nueva clave generada con éxito!')
      ),

      // Sección de API Keys
      h('div', { class: 'bg-white shadow overflow-hidden sm:rounded-lg' },
        h('div', { class: 'px-4 py-5 sm:px-6 border-b border-gray-200' },
          h('h2', { class: 'text-lg font-medium text-gray-900' }, 'Tus claves API')
        ),
        h('div', { class: 'divide-y divide-gray-200' },
          apiKeys.length > 0 
            ? apiKeys.map(apiKey => h(ApiKeyCard, {
                key: apiKey.id,
                apiKey: apiKey.key,
                onCopy: () => handleCopyKey(apiKey.key),
                onRevoke: () => handleRevokeKey(apiKey.id)
              }))
            : h('div', { class: 'p-6 text-center text-gray-500' },
                'No hay claves API. Crea una nueva para comenzar.'
              )
        )
      )
    ),

    // Modal para crear nueva clave
    h(CreateKeyModal, {
      isOpen: showCreateModal,
      onClose: () => setShowCreateModal(false),
      onCreate: handleCreateKey
    })
  );
}

// Renderizar la aplicación
render(h(Dashboard), document.getElementById('app'));
