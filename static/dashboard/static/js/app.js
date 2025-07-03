// Usar Preact desde el ámbito global
const { h, render } = window.preact || window;
const { useState, useEffect } = window.preactHooks || window;

// Función para mostrar notificaciones
function showNotification(message, type = 'success') {
  const toast = document.createElement('div');
  const bgColor = type === 'error' ? 'bg-red-500' : 'bg-green-500';
  
  toast.className = `fixed bottom-4 right-4 ${bgColor} text-white px-6 py-3 font-bold text-sm border-2 border-black shadow-[4px_4px_0_#000] uppercase tracking-wider`;
  toast.textContent = message;
  
  document.body.appendChild(toast);
  
  setTimeout(() => {
    toast.style.opacity = '0';
    setTimeout(() => toast.remove(), 200);
  }, 3000);
}

// Componente para mostrar el valor de la API Key
function ApiKeyValue({ value, isVisible, onToggle }) {
  return h('div', { 
    className: 'relative group',
    style: { minHeight: '60px' }
  },
    [
      h('div', { 
        className: 'font-mono text-sm p-3',
        style: { 
          border: '2px solid #000',
          backgroundColor: '#fff',
          wordBreak: 'break-all',
          filter: isVisible ? 'none' : 'blur(4px)',
          transition: 'filter 0.2s ease',
          minHeight: '60px',
          display: 'flex',
          alignItems: 'center',
          lineHeight: '1.4'
        }
      }, isVisible ? value : '•'.repeat(32)),
      
      h('button', {
        onClick: (e) => {
          e.stopPropagation();
          onToggle();
        },
        className: 'absolute right-2 top-1/2 transform -translate-y-1/2 bg-white',
        title: isVisible ? 'Ocultar clave' : 'Mostrar clave',
        type: 'button',
        style: {
          width: '28px',
          height: '28px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: '2px solid #000',
          boxShadow: '2px 2px 0 #000'
        }
      }, 
        h('i', { 
          className: `fas ${isVisible ? 'fa-eye-slash' : 'fa-eye'}`,
          'aria-hidden': 'true',
          style: { 
            fontSize: '14px',
            color: '#000'
          }
        })
      )
    ]
  );
}

// Componente para la lista de claves API
function ApiKeyList({ apiKeys, loadApiKeys }) {
  const [visibleKeys, setVisibleKeys] = useState({});
  
  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      showNotification('¡Clave copiada al portapapeles!');
    });
  };
  
  const revokeKey = async (keyId) => {
    if (!confirm('¿Estás seguro de que deseas revocar esta clave? Esta acción no se puede deshacer.')) {
      return;
    }
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/keys/${keyId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (response.ok) {
        showNotification('Clave revocada correctamente');
        loadApiKeys();
      } else {
        const error = await response.json();
        showNotification(`Error al revocar la clave: ${error.message || 'Error desconocido'}`, 'error');
      }
    } catch (error) {
      console.error('Error al revocar la clave:', error);
      showNotification('Error al conectar con el servidor', 'error');
    }
  };
  
  if (apiKeys.length === 0) {
    return h('div', { className: 'text-center py-8' },
      h('p', { className: 'text-gray-500' }, 'No hay claves API generadas')
    );
  }
  
  return h('div', { className: 'space-y-4' },
    apiKeys.map(apiKey => {
      const isVisible = visibleKeys[apiKey.id] || false;
      
      return h('div', { 
        key: apiKey.id,
        className: 'neo-card p-4 bg-white'
      },
        [
          h('div', { className: 'flex justify-between items-start mb-3' },
            h('div', null,
              h('h3', { className: 'font-bold text-lg' }, apiKey.name || 'Sin nombre')
            ),
            h('div', { className: 'flex space-x-2' },
              h('button', {
                onClick: () => copyToClipboard(apiKey.key),
                className: 'px-3 py-1 text-sm border border-black bg-white hover:bg-gray-100',
                title: 'Copiar clave'
              }, '📋 Copiar'),
              h('button', {
                onClick: () => revokeKey(apiKey.id),
                className: 'px-3 py-1 text-sm border border-black bg-red-100 hover:bg-red-200',
                title: 'Revocar clave'
              }, '🗑️ Eliminar')
            )
          ),
          
          h('div', { className: 'mb-3' },
            h('p', { className: 'text-sm text-gray-600 mb-1' }, 'ID: ' + apiKey.id)
          ),
          
          h('div', { className: 'mb-3' },
            h('p', { className: 'text-sm text-gray-600 mb-1' }, 'Creada: ' + new Date(apiKey.created_at).toLocaleString())
          ),
          
          h('div', { className: 'relative' },
            h('label', { className: 'block text-sm font-medium text-gray-700 mb-1' }, 'Clave API:'),
            h(ApiKeyValue, {
              value: apiKey.key,
              isVisible,
              onToggle: () => setVisibleKeys(prev => ({
                ...prev,
                [apiKey.id]: !prev[apiKey.id]
              }))
            })
          )
        ]
      );
    })
  );
}

// Componente principal de la aplicación
function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [userEmail, setUserEmail] = useState('');
  const [apiKeys, setApiKeys] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyValue, setNewKeyValue] = useState('');
  const [showNewKeyModal, setShowNewKeyModal] = useState(false);
  const [isKeyVisible, setIsKeyVisible] = useState(false);
  const [error, setError] = useState('');
  
  // Cargar claves API al montar el componente
  useEffect(() => {
    loadApiKeys();
  }, []);
  
  // Función para cargar las claves API
  const loadApiKeys = async () => {
    setIsLoading(true);
    setError('');
    try {
      const token = localStorage.getItem('token');
      if (!token) {
        // Si no hay token, redirigir al login
        window.location.href = '/';
        return;
      }
      
      const response = await fetch('/api/keys', {
        headers: { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Accept': 'application/json'
        },
        credentials: 'include'
      });
      
      console.log('Respuesta de /api/keys:', response.status, response.statusText);
      
      if (response.ok) {
        const data = await response.json();
        console.log('Datos de claves API recibidos:', data);
        // Asegurarse de que data.api_keys sea un array
        const keysArray = Array.isArray(data.api_keys) ? data.api_keys : [];
        console.log('Claves API procesadas:', keysArray);
        setApiKeys(keysArray);
      } else {
        console.error('Error en la respuesta del servidor:', response.status, await response.text());
        setApiKeys([]);
      }
    } catch (error) {
      console.error('Error al cargar claves API:', error);
      setApiKeys([]);
    } finally {
      setIsLoading(false);
    }
  };
  
  // Verificar autenticación al cargar
  useEffect(() => {
    const checkAuth = async () => {
      const token = localStorage.getItem('token');
      const email = localStorage.getItem('userEmail');
      
      if (token && email) {
        setIsAuthenticated(true);
        setUserEmail(email);
      } else {
        // Si no hay token, redirigir al login
        window.location.href = '/';
      }
    };
    
    checkAuth();
  }, []);
  
  // Generar una nueva clave API
  const generateApiKey = async () => {
    if (!newKeyName.trim()) {
      alert('Por favor ingresa un nombre para la clave');
      return;
    }
    
    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ name: newKeyName })
      });
      
      if (response.ok) {
        const data = await response.json();
        console.log('Clave API generada:', data);
        // Asegurarse de que la clave esté en el formato correcto
        const apiKey = data.key || data.api_key;
        if (!apiKey) {
          throw new Error('No se recibió una clave API válida');
        }
        setNewKeyValue(apiKey);
        setShowNewKeyModal(true);
        setNewKeyName('');
        // Recargar la lista de claves después de un breve retraso
        setTimeout(loadApiKeys, 500);
      } else {
        const errorData = await response.json();
        alert(errorData.message || 'Error al generar la clave API');
      }
    } catch (error) {
      console.error('Error al generar clave API:', error);
      alert('Error al generar la clave API');
    }
  };
  
  // Cerrar sesión
  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('userEmail');
    setIsAuthenticated(false);
    setUserEmail('');
    setApiKeys([]);
    // Redirigir al login
    window.location.href = '/';
  };
  
  // Renderizado condicional basado en el estado de carga
  if (isLoading) {
    return h('div', { className: 'min-h-screen flex items-center justify-center' },
      h('div', { className: 'animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-black' })
    );
  }
  
  // Estilos globales inyectados
  useEffect(() => {
    const style = document.createElement('style');
    style.textContent = `
      .neo-card {
        background: #FFFFFF;
        border: 2px solid #000000;
        border-radius: 0.5rem;
        box-shadow: 4px 4px 0 #000000;
        transition: all 0.2s ease;
      }
      .neo-card:hover {
        transform: translate(-2px, -2px);
        box-shadow: 6px 6px 0 #000000;
      }
      .neo-btn {
        background: #4F46E5;
        color: white;
        border: 2px solid #000000;
        border-radius: 0.375rem;
        box-shadow: 3px 3px 0 #000000;
        transition: all 0.2s ease;
      }
      .neo-btn:hover:not(:disabled) {
        transform: translate(-1px, -1px);
        box-shadow: 4px 4px 0 #000000;
      }
      .neo-btn:active:not(:disabled) {
        transform: translate(1px, 1px);
        box-shadow: 2px 2px 0 #000000;
      }
      .neo-btn:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }
    `;
    document.head.appendChild(style);
    return () => document.head.removeChild(style);
  }, []);
  
  return h('div', { className: 'min-h-screen bg-gray-50' },
    [
      // Header
      h('header', { className: 'bg-white shadow-sm' },
        h('div', { className: 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4' },
          h('div', { className: 'flex justify-between items-center' },
            [
              h('h1', { className: 'text-2xl font-bold text-gray-900' }, 'Toolbox API'),
              isAuthenticated ? h('div', { className: 'flex items-center space-x-4' },
                [
                  h('a', {
                    href: '/docs',
                    target: '_blank',
                    className: 'text-sm font-medium text-indigo-600 hover:text-indigo-800 hover:underline',
                    title: 'Documentación de la API'
                  }, 'Documentación'),
                  h('span', { className: 'text-sm font-medium text-gray-800' }, `Hola, ${userEmail}`),
                  h('button', {
                    onClick: handleLogout,
                    className: 'px-4 py-2 bg-red-100 text-red-800 border-2 border-black shadow-[3px_3px_0_#000] hover:bg-red-200 transition-all text-sm font-medium'
                  }, 'Cerrar sesión')
                ]
              ) : null
            ]
          )
        )
      ),
      
      // Main Content
      h('main', { className: 'max-w-4xl mx-auto px-4 py-8' },
        [
          // API Keys List Section
          h('div', { className: 'mb-8' },
            [
              h('div', { className: 'flex justify-between items-center mb-6' },
                [
                  h('h2', { className: 'text-2xl font-bold text-gray-900' }, 'Tus claves API'),
                  h('button', {
                    onClick: () => setShowCreateModal(true),
                    className: 'neo-btn px-4 py-2 font-medium',
                    disabled: isLoading
                  }, 'Nueva clave')
                ]
              ),
              h('div', { className: 'space-y-4' },
                h(ApiKeyList, { apiKeys, loadApiKeys })
              )
            ]
          )
        ]
      ),
      
      // Modal para crear nueva clave
      showCreateModal && h('div', { className: 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50' },
        h('div', { className: 'bg-white p-6 rounded-lg shadow-xl max-w-md w-full' },
          [
            h('h2', { className: 'text-xl font-bold mb-4' }, 'Nueva clave API'),
            h('div', { className: 'mb-4' },
              h('label', { 
                htmlFor: 'keyName',
                className: 'block text-sm font-medium text-gray-700 mb-1'
              }, 'Nombre de la clave'),
              h('input', {
                type: 'text',
                id: 'keyName',
                value: newKeyName,
                onChange: (e) => setNewKeyName(e.target.value),
                className: 'w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500',
                placeholder: 'Ej: Aplicación móvil'
              })
            ),
            h('div', { className: 'flex justify-end space-x-3' },
              [
                h('button', {
                  onClick: () => setShowCreateModal(false),
                  className: 'px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50',
                  type: 'button'
                }, 'Cancelar'),
                h('button', {
                  onClick: generateApiKey,
                  className: 'px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700',
                  type: 'button',
                  disabled: !newKeyName.trim()
                }, 'Generar clave')
              ]
            )
          ]
        )
      ),
      
      // Modal para mostrar la nueva clave generada
      showNewKeyModal && h('div', { className: 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50' },
        h('div', { className: 'bg-white p-6 rounded-lg shadow-xl max-w-md w-full' },
          [
            h('h2', { className: 'text-xl font-bold mb-4' }, '¡Clave generada con éxito!'),
            h('p', { className: 'text-sm text-gray-600 mb-4' }, 'Guarda esta clave en un lugar seguro. No podrás verla de nuevo.'),
            h('div', { className: 'mb-6' },
              h('label', { 
                className: 'block text-sm font-medium text-gray-700 mb-1'
              }, 'Tu clave API:'),
              h('div', { className: 'relative' },
                h('input', {
                  type: isKeyVisible ? 'text' : 'password',
                  value: newKeyValue,
                  readOnly: true,
                  className: 'w-full px-3 py-2 pr-10 border border-gray-300 rounded-md font-mono text-sm',
                  style: { userSelect: 'all' }
                }),
                h('button', {
                  onClick: () => setIsKeyVisible(!isKeyVisible),
                  className: 'absolute inset-y-0 right-0 px-3 flex items-center',
                  type: 'button',
                  title: isKeyVisible ? 'Ocultar' : 'Mostrar'
                },
                  h('i', { 
                    className: `fas ${isKeyVisible ? 'fa-eye-slash' : 'fa-eye'} text-gray-500`,
                    'aria-hidden': 'true'
                  })
                )
              )
            ),
            h('div', { className: 'flex justify-end' },
              h('button', {
                onClick: () => {
                  setShowNewKeyModal(false);
                  setIsKeyVisible(false);
                },
                className: 'px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700',
                type: 'button'
              }, 'Entendido')
            )
          ]
        )
      )
    ]
  );
}

// Renderizar la aplicación
const appContainer = document.getElementById('app');
if (appContainer) {
  render(h(App), appContainer);
}
