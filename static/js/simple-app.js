// Usar Preact desde el ámbito global
const { h, render } = window.preact || window;
const { useState, useEffect } = window.preactHooks || window;

// Componente de modal para login con magic link
function LoginModal({ isOpen, onClose, onSuccess }) {
  const [email, setEmail] = useState('');
  const [message, setMessage] = useState('');
  const [isError, setIsError] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    try {
      const response = await fetch('/api/auth/request-magic-link', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email })
      });

      const data = await response.json();
      
      if (response.ok) {
        setMessage('¡Revisa tu correo electrónico para el enlace de inicio de sesión!');
        setIsError(false);
        setEmail('');
        
        // Cerrar el modal después de 3 segundos
        setTimeout(() => {
          onClose();
          setMessage('');
        }, 3000);
      } else {
        throw new Error(data.message || 'Error al enviar el enlace mágico');
      }
    } catch (error) {
      setMessage(error.message || 'Error al enviar el enlace mágico');
      setIsError(true);
    }
  };

  if (!isOpen) return null;

  return h('div', { class: 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50' },
    h('div', { class: 'bg-white rounded-lg shadow-xl w-full max-w-md' },
      h('div', { class: 'p-6' },
        h('div', { class: 'flex justify-between items-center mb-4' },
          h('h3', { class: 'text-lg font-medium text-gray-900' }, 'Iniciar sesión'),
          h('button', {
            onClick: onClose,
            class: 'text-gray-400 hover:text-gray-500',
            'aria-label': 'Cerrar'
          }, '×')
        ),
        
        message && h('div', {
          class: `p-3 mb-4 rounded ${isError ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`
        }, message),
        
        h('form', { onSubmit: handleSubmit },
          h('div', { class: 'mb-4' },
            h('label', {
              for: 'email',
              class: 'block text-sm font-medium text-gray-700 mb-1'
            }, 'Correo electrónico'),
            h('input', {
              type: 'email',
              id: 'email',
              value: email,
              onInput: (e) => setEmail(e.target.value),
              class: 'w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500',
              placeholder: 'tucorreo@ejemplo.com',
              required: true
            })
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
            }, 'Enviar enlace mágico')
          )
        )
      )
    )
  );
}

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
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [newKey, setNewKey] = useState(null);

  // Verificar autenticación y cargar claves API al iniciar
  useEffect(() => {
    const checkAuth = async () => {
      try {
        const response = await fetch('/api/auth/me', {
          method: 'GET',
          credentials: 'include',
          headers: {
            'Accept': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          }
        });
        
        if (response.ok) {
          const userData = await response.json();
          console.log('Usuario autenticado:', userData.email);
          setIsAuthenticated(true);
          await loadApiKeys();
        } else {
          console.log('No autenticado o error en la autenticación');
          setIsAuthenticated(false);
        }
      } catch (error) {
        console.error('Error al verificar autenticación:', error);
        setIsAuthenticated(false);
      } finally {
        setIsLoading(false);
      }
    };

    const loadApiKeys = async () => {
      try {
        const response = await fetch('/api/keys', {
          method: 'GET',
          credentials: 'include',
          headers: {
            'Accept': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          }
        });
        
        if (!response.ok) {
          throw new Error('Error al cargar las claves API');
        }
        
        const data = await response.json();
        // The backend returns {success: true, keys: [...]}
        setApiKeys(data.keys || []);
      } catch (error) {
        console.error('Error al cargar las claves API:', error);
        throw error; // Propagar el error para manejarlo en checkAuth
      }
    };

    checkAuth();
  }, []);

  const handleCreateKey = async (keyData) => {
    try {
      const response = await fetch('/api/keys', {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        body: JSON.stringify({
          name: keyData.name || 'Nueva clave',
          permissions: Object.entries(keyData.permissions || {})
            .filter(([_, value]) => value)
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

  const handleCreateClick = () => {
    if (isAuthenticated) {
      setShowCreateModal(true);
    } else {
      setShowLoginModal(true);
    }
  };

  const handleLoginSuccess = () => {
    setShowLoginModal(false);
    setIsAuthenticated(true);
    // Recargar las claves después de iniciar sesión
    loadApiKeys();
  };

  const handleRevokeKey = async (keyId) => {
    if (!confirm('¿Estás seguro de que deseas revocar esta clave? Esta acción no se puede deshacer.')) {
      return;
    }

    try {
      const response = await fetch(`/api/keys/${keyId}`, {
        method: 'DELETE',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        body: JSON.stringify({ key_id: keyId })
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

  // Mostrar loading
  if (isLoading) {
    return h('div', { class: 'flex justify-center items-center h-64' },
      h('div', { class: 'animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500' })
    );
  }
  
  // Mostrar botón de login si no está autenticado
  if (!isAuthenticated) {
    return h('div', { class: 'max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8' },
      h('div', { class: 'bg-white overflow-hidden shadow rounded-lg' },
        h('div', { class: 'px-4 py-5 sm:p-6 text-center' },
          h('h2', { class: 'text-2xl font-bold text-gray-900 mb-4' }, 'Acceso requerido'),
          h('p', { class: 'text-gray-600 mb-6' }, 'Por favor inicia sesión para acceder al panel de control.'),
          h('button', {
            onClick: () => setShowLoginModal(true),
            class: 'px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
          }, 'Iniciar sesión')
        )
      )
    );
  }

  return h('div', { class: 'min-h-screen bg-gray-100' },
    // Navbar
    h('nav', { class: 'bg-white shadow-sm' },
      h('div', { class: 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8' },
        h('div', { class: 'flex justify-between h-16' },
          h('div', { class: 'flex' },
            h('div', { class: 'flex-shrink-0 flex items-center' },
              h('a', { 
                href: '/',
                class: 'px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-100'
              }, '← Volver al inicio')
            )
          ),
          isAuthenticated && h('div', { class: 'flex items-center' },
            h('a', {
              href: '/api/auth/logout',
              class: 'text-sm text-gray-500 hover:text-gray-700',
              onClick: (e) => {
                if (!confirm('¿Estás seguro de que quieres cerrar sesión?')) {
                  e.preventDefault();
                }
              }
            }, 'Cerrar sesión')
          )
        )
      )
    ),

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

      // Sección de herramientas
      h('div', { class: 'mb-8' },
        h('h2', { class: 'text-lg font-medium text-gray-900 mb-4' }, 'Herramientas disponibles'),
        h('div', { class: 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6' },
          // Tarjeta de Screenshot
          h('div', { class: 'bg-white overflow-hidden shadow rounded-lg' },
            h('div', { class: 'p-6' },
              h('div', { class: 'flex items-center mb-4' },
                h('div', { class: 'flex-shrink-0 bg-indigo-100 p-3 rounded-md' },
                  h('i', { class: 'fas fa-camera text-indigo-600 text-xl' })
                ),
                h('h3', { class: 'ml-3 text-lg font-medium text-gray-900' }, 'Screenshot')
              ),
              h('p', { class: 'text-gray-600 mb-4' }, 'Toma capturas de pantalla de cualquier página web. Perfecto para generar vistas previas o documentación.'),
              h('div', { class: 'flex justify-between items-center' },
                h('a', {
                  href: '/docs/screenshot',
                  target: '_blank',
                  class: 'text-sm font-medium text-indigo-600 hover:text-indigo-500',
                  title: 'Ver documentación'
                }, 'Documentación'),
                h('a', {
                  href: '/screenshot-tester',
                  class: 'px-3 py-1.5 bg-indigo-600 text-white text-sm font-medium rounded-md hover:bg-indigo-700',
                  title: 'Probar herramienta'
                }, 'Probar')
              )
            )
          )
        )
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

    // Renderizar el modal de login
    showLoginModal && h(LoginModal, {
      isOpen: true,
      onClose: () => setShowLoginModal(false),
      onSuccess: handleLoginSuccess
    }),

    // Renderizar el modal de crear clave
    showCreateModal && h(CreateKeyModal, {
      isOpen: true,
      onClose: () => setShowCreateModal(false),
      onCreate: handleCreateKey
    })
  );
}

// Hacer que la función Dashboard esté disponible globalmente
window.Dashboard = Dashboard;

// Renderizar la aplicación si el elemento #app existe
if (document.getElementById('app')) {
  render(h(Dashboard), document.getElementById('app'));
}
