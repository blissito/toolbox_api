// Usar Preact desde el ámbito global
const { h, render } = window.preact || window;
const { useState, useEffect } = window.preactHooks || window;

// Helper function to create elements
function createElement(type, props, ...children) {
  return h(type, props, ...children);
}

// Components
function ApiKeyList({ apiKeys, loadApiKeys }) {
  // Asegurarse de que apiKeys sea un array
  const keys = Array.isArray(apiKeys) ? apiKeys : [];
  
  if (keys.length === 0) {
    return h('div', { className: 'text-center py-12' },
      h('p', { className: 'text-gray-600' }, 'No hay claves API generadas.'),
      h('p', { className: 'text-gray-500 text-sm mt-1' }, 'Crea una nueva para comenzar.')
    );
  }

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      // Mostrar un mensaje más sutil
      const toast = document.createElement('div');
      toast.className = 'fixed bottom-4 right-4 bg-gray-900 text-white px-4 py-2 rounded-md shadow-lg font-medium text-sm';
      toast.textContent = '¡Clave copiada al portapapeles!';
      document.body.appendChild(toast);
      setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 200);
      }, 2000);
    });
  };

  // Función para formatear fechas de manera más concisa
  const formatDate = (dateString) => {
    try {
      const date = new Date(dateString);
      return isNaN(date.getTime()) ? '' : date.toLocaleDateString('es-MX', {
        day: '2-digit',
        month: 'short',
        year: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch (e) {
      console.error('Error al formatear fecha:', e);
      return '';
    }
  };

  // Función para manejar la revocación de una clave
  const handleRevokeKey = async (keyId, keyName) => {
    if (!confirm(`¿Revocar "${keyName || 'clave sin nombre'}"?`)) {
      return;
    }

    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/keys/revoke/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ key_id: keyId })
      });

      const result = await response.json();
      
      if (response.ok) {
        loadApiKeys(); // Recargar la lista
      } else {
        alert(`Error: ${result.error || 'Error desconocido'}`);
      }
    } catch (error) {
      console.error('Error al revocar:', error);
      alert('Error de conexión');
    }
  };

  return h('div', { className: 'space-y-4' },
    keys.map(key => {
      return h('div', { 
        key: key.id,
        className: 'neo-card p-5 transition-all duration-200 hover:shadow-md'
      },
        [
          // Header con nombre y estado
          h('div', { className: 'flex justify-between items-start mb-3' },
            [
              h('h3', { 
                className: `text-lg font-semibold ${key.revoked ? 'text-gray-500' : 'text-gray-900'}`
              }, key.name || 'Clave sin nombre'),
              h('span', {
                className: `px-2 py-1 text-xs font-medium rounded ${key.revoked ? 'bg-gray-100 text-gray-600' : 'bg-green-100 text-green-800'}`
              }, key.revoked ? 'Revocada' : 'Activa')
            ]
          ),
          
          // Valor de la clave
          h('div', { className: 'mb-4' },
            h('div', { className: 'bg-gray-50 p-3 rounded-md' },
              h('code', { 
                className: 'font-mono text-sm text-gray-800 break-all',
                style: 'word-break: break-all; display: block;'
              }, key.key_value || '••••••••••••••••••••••••••••••••')
            )
          ),
          
          // Footer con fecha y acciones
          h('div', { className: 'flex justify-between items-center pt-3 border-t border-gray-100' },
            [
              h('span', { 
                className: 'text-xs text-gray-500',
                title: `Creada el ${new Date(key.created_at).toLocaleString()}`
              }, `Creada el ${formatDate(key.created_at)}`),
              
              !key.revoked && h('button', {
                onClick: () => handleRevokeKey(key.id, key.name),
                className: 'text-sm px-3 py-1 bg-red-50 hover:bg-red-100 text-red-600 rounded-md border border-red-200',
                title: 'Revocar clave'
              }, 'Revocar')
            ]
          )
        ]
      );
    })
  );
}

function LoginModal({ onClose, onLogin }) {
  const [email, setEmail] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!email) {
      setError('Por favor ingresa tu correo electrónico');
      return;
    }
    
    setIsLoading(true);
    setError('');
    
    try {
      const response = await fetch('/api/auth/request-magic-link', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email })
      });
      
      if (response.ok) {
        alert('¡Revisa tu correo para el enlace de inicio de sesión!');
        onClose();
      } else {
        const errorData = await response.json();
        setError(errorData.message || 'Error al enviar el enlace de inicio de sesión');
      }
    } catch (err) {
      setError('Error de conexión. Intenta de nuevo más tarde.');
      console.error('Login error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // Estructura del modal de login
  return h('div', {
    className: 'fixed inset-0 bg-black bg-opacity-75 backdrop-blur-sm flex items-center justify-center p-4 z-50 transition-opacity duration-300'
  }, h('div', {
    className: 'bg-gradient-to-br from-darker-bg to-card-bg rounded-xl p-8 w-full max-w-md border border-opacity-20 border-neon-blue shadow-2xl transform transition-all duration-300'
  }, [
    // Encabezado
    h('div', {
      className: 'flex justify-between items-center mb-6'
    }, [
      h('h2', {
        className: 'text-2xl font-bold bg-gradient-to-r from-neon-pink via-neon-purple to-neon-blue bg-clip-text text-transparent tracking-tight'
      }, 'Iniciar sesión'),
      h('button', {
        onClick: onClose,
        type: 'button',
        className: 'text-gray-400 hover:text-neon-pink transition-colors duration-200 text-xl p-1 -mr-2'
      }, '✕')
    ]),
    
    // Descripción
    h('p', {
      className: 'text-gray-400 text-sm mb-6'
    }, 'Ingresa tu correo electrónico para recibir un enlace mágico de inicio de sesión.'),
    
    // Mensaje de error si existe
    error ? h('div', {
      className: 'bg-red-900/50 border border-red-700 text-red-100 p-3 rounded-lg mb-6 text-sm flex items-center'
    }, [
      h('span', { className: 'mr-2' }, '⚠️'),
      error
    ]) : null,
    
    // Formulario
    h('form', {
      onSubmit: handleSubmit,
      className: 'space-y-6'
    }, [
      // Campo de email
      h('div', {
        className: 'space-y-2'
      }, [
        h('label', {
          htmlFor: 'email',
          className: 'block text-sm font-medium text-neon-blue/90'
        }, 'Correo electrónico'),
        h('input', {
          type: 'email',
          id: 'email',
          value: email,
          onInput: (e) => setEmail(e.target.value),
          className: 'w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-neon-blue',
          placeholder: 'tu@email.com',
          disabled: isLoading
        })
      ]),
      
      // Botón de envío
      h('button', {
        type: 'submit',
        disabled: isLoading,
        className: 'w-full py-2 px-4 bg-gradient-to-r from-neon-purple to-neon-pink text-white rounded-md font-medium hover:opacity-90 transition-opacity disabled:opacity-50 flex items-center justify-center'
      }, isLoading ? 'Enviando...' : 'Enviar enlace mágico')
    ])
  ]));
}

function ApiKeyModal({ apiKey, onClose }) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(apiKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return h('div', { 
    className: 'fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 z-50 transition-all duration-300',
    style: { animation: 'fadeIn 0.3s ease-out' }
  }, h('div', { 
    className: 'bg-gradient-to-br from-gray-900 via-gray-900/95 to-gray-900/90 rounded-2xl border border-neon-blue/20 shadow-2xl p-6 w-full max-w-md transform transition-all duration-300',
    style: { 
      boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.6), 0 0 15px -3px rgba(59, 130, 246, 0.2)',
      border: '1px solid rgba(0, 242, 254, 0.2)',
      animation: 'slideUp 0.4s cubic-bezier(0.16, 1, 0.3, 1)'
    }
  }, [
    // Encabezado
    h('div', { 
      className: 'flex justify-between items-start mb-6'
    }, [
      h('div', {},
        h('h2', { 
          className: 'text-2xl font-bold bg-gradient-to-r from-neon-pink via-neon-purple to-neon-blue bg-clip-text text-transparent tracking-tight leading-tight mb-1'
        }, '¡Clave API generada!'),
        h('p', {
          className: 'text-sm text-gray-400 mt-1'
        }, 'Tu nueva clave de acceso está lista')
      ),
      h('button', { 
        onClick: onClose, 
        type: 'button',
        className: 'text-gray-500 hover:text-neon-pink transition-all duration-200 text-xl p-1 -mt-1 -mr-2 hover:scale-110 transform'
      }, '✕')
    ]),
    
    // Clave API
    h('div', { 
      className: 'relative group mb-6'
    },
      h('div', {
        className: 'relative bg-gray-900/80 border border-neon-blue/20 rounded-xl p-4 font-mono text-sm text-neon-green/90 break-all transition-all duration-300 hover:border-neon-blue/50 hover:shadow-lg hover:shadow-neon-blue/10',
        style: {
          backdropFilter: 'blur(4px)',
          wordBreak: 'break-all'
        }
      },
        h('div', { 
          className: 'absolute -top-2 -right-2 bg-neon-blue text-gray-900 text-xs font-bold px-2 py-0.5 rounded-full shadow-md',
          style: { zIndex: 1 }
        }, 'NUEVA'),
        h('div', { 
          className: 'relative z-10',
          style: { 
            filter: 'drop-shadow(0 0 8px rgba(0, 242, 254, 0.3))',
            textShadow: '0 0 8px rgba(0, 242, 254, 0.5)'
          }
        }, apiKey)
      )
    ),
    
    // Mensaje
    h('div', { 
      className: 'bg-blue-900/10 border border-blue-900/20 rounded-lg p-3 mb-6 text-center'
    },
      h('p', { 
        className: 'text-sm text-blue-200/90 flex items-center justify-center gap-2'
      }, [
        h('span', { className: 'text-yellow-400' }, '⚠️'),
        '¡Guarda esta clave en un lugar seguro! No podrás volver a verla.'
      ])
    ),
    
    // Botones
    h('div', { 
      className: 'flex flex-col sm:flex-row gap-3'
    }, [
      h('button', {
        onClick: copyToClipboard,
        className: 'group flex-1 py-3 px-6 bg-gradient-to-r from-neon-purple to-neon-pink text-white rounded-xl font-medium hover:opacity-90 transition-all duration-200 flex items-center justify-center gap-2 relative overflow-hidden',
        type: 'button',
        style: {
          boxShadow: '0 4px 15px -5px rgba(255, 46, 99, 0.4)'
        }
      }, [
        h('span', { 
          className: 'relative z-10 flex items-center gap-2',
          style: { textShadow: '0 1px 2px rgba(0,0,0,0.2)' }
        },
          copied ? '✓ ¡Copiada!' : '📋 Copiar clave',
          copied && h('span', { className: 'text-xs opacity-90' }, '✓')
        ),
        h('span', {
          className: 'absolute inset-0 bg-white/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300',
          style: {
            background: 'linear-gradient(135deg, rgba(255,255,255,0.15) 0%, rgba(255,255,255,0) 100%)'
          }
        })
      ]),
      
      h('button', {
        onClick: onClose,
        className: 'flex-1 py-3 px-6 bg-gray-800/30 border border-gray-700/50 text-white rounded-xl font-medium hover:bg-gray-700/40 hover:border-neon-blue/40 transition-all duration-200 hover:shadow-lg hover:shadow-neon-blue/5',
        type: 'button',
        style: {
          backdropFilter: 'blur(4px)'
        }
      }, 'Entendido')
    ]),
    
    // Estilos de animación
    h('style', null, `
      @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
      }
      @keyframes slideUp {
        from { 
          opacity: 0;
          transform: translateY(20px);
        }
        to { 
          opacity: 1;
          transform: translateY(0);
        }
      }
    `)
  ]));
}

// Componente principal de la aplicación
function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [userEmail, setUserEmail] = useState('');
  const [apiKeys, setApiKeys] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [showKeyModal, setShowKeyModal] = useState(false);
  const [newApiKey, setNewApiKey] = useState('');
  
  // Asegurarse de que los estilos globales estén aplicados
  useEffect(() => {
    const style = document.createElement('style');
    style.textContent = `
      .neo-card {
        border: 2px solid #000;
        box-shadow: 6px 6px 0 #000;
        transition: all 0.2s ease;
      }
      .neo-card:hover {
        transform: translate(3px, 3px);
        box-shadow: 3px 3px 0 #000;
      }
      .neo-btn {
        background: white;
        border: 2px solid #000;
        box-shadow: 4px 4px 0 #000;
        transition: all 0.2s ease;
      }
      .neo-btn:hover {
        transform: translate(2px, 2px);
        box-shadow: 2px 2px 0 #000;
      }
    `;
    document.head.appendChild(style);
    
    return () => {
      document.head.removeChild(style);
    };
  }, []);

  // Cargar claves API
  const loadApiKeys = async () => {
    if (!isAuthenticated) return;
    
    try {
      console.log('Cargando claves API...');
      const token = localStorage.getItem('token');
      if (!token) {
        console.error('No hay token de autenticación');
        return;
      }
      
      const response = await fetch('/api/keys/list', {
        headers: { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Accept': 'application/json'
        },
        credentials: 'include'
      });
      
      console.log('Respuesta de /api/keys/list:', response.status, response.statusText);
      
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
    }
  };

  // Verificar autenticación al cargar
  useEffect(() => {
    const checkAuth = async () => {
      console.log('Verificando autenticación...');
      // Verificar si hay un token en la URL (redirección después del login)
      const urlParams = new URLSearchParams(window.location.search);
      const tokenFromUrl = urlParams.get('token');
      
      if (tokenFromUrl) {
        console.log('Token encontrado en URL');
        // Limpiar la URL
        window.history.replaceState({}, document.title, window.location.pathname);
        localStorage.setItem('token', tokenFromUrl);
      }

      const token = localStorage.getItem('token');
      console.log('Token en localStorage:', token ? 'presente' : 'ausente');
      
      if (!token) {
        console.log('No hay token, mostrando modal de login');
        setIsAuthenticated(false);
        setIsLoading(false);
        setShowLoginModal(true);
        return;
      }

      try {
        console.log('Verificando token con el servidor...');
        const response = await fetch('/api/auth/me', {
          headers: { 
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
            'Accept': 'application/json'
          },
          credentials: 'include'
        });

        console.log('Respuesta de /api/auth/me:', response.status);
        
        if (response.ok) {
          const userData = await response.json();
          console.log('Usuario autenticado:', userData.email);
          setUserEmail(userData.email || 'Usuario');
          setIsAuthenticated(true);
          // Cargar las claves después de autenticarse
          await loadApiKeys();
          setShowLoginModal(false);
        } else {
          console.error('Error en la autenticación, eliminando token');
          localStorage.removeItem('token');
          setIsAuthenticated(false);
          setShowLoginModal(true);
        }
      } catch (error) {
        console.error('Error al verificar autenticación:', error);
        localStorage.removeItem('token');
        setIsAuthenticated(false);
        setShowLoginModal(true);
      } finally {
        setIsLoading(false);
      }
    };

    checkAuth();
  }, []);
  
  // Cargar claves cuando cambie el estado de autenticación
  useEffect(() => {
    if (isAuthenticated) {
      console.log('Usuario autenticado, cargando claves...');
      loadApiKeys();
    } else {
      console.log('Usuario no autenticado, limpiando claves');
      setApiKeys([]);
    }
  }, [isAuthenticated]);

  // La función loadApiKeys ya está definida al inicio del componente

  // Generar una nueva clave API
  const generateApiKey = async () => {
    if (!isAuthenticated) {
      setShowLoginModal(true);
      return;
    }

    try {
      const token = localStorage.getItem('token');
      const response = await fetch('/api/keys/create', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          name: `Clave ${new Date().toLocaleDateString()}`
        })
      });

      if (response.ok) {
        const data = await response.json();
        console.log('Clave API generada:', data); // Para depuración
        // Asegurarse de que la clave esté en el formato correcto
        const apiKey = data.key || data.api_key;
        if (!apiKey) {
          throw new Error('No se recibió una clave API válida');
        }
        setNewApiKey(apiKey);
        setShowKeyModal(true);
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
    setShowLoginModal(true); // Mostrar modal de login al cerrar sesión
  };

  // Copiar al portapapeles
  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      alert('¡Clave copiada al portapapeles!');
    }).catch(err => {
      console.error('Error al copiar:', err);
      alert('No se pudo copiar la clave');
    });
  };

  // Renderizado condicional basado en el estado de carga
  if (isLoading) {
    return h('div', { className: 'min-h-screen flex items-center justify-center' },
      h('div', { className: 'animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-neon-blue' })
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
      h('header', { className: 'bg-white border-b-2 border-black' },
        h('div', { className: 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-between items-center' },
          [
            h('h1', { 
              className: 'text-2xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent'
            }, 'Toolbox API'),
            
            isAuthenticated && h('div', { className: 'flex items-center space-x-6' },
              [
                h('a', { 
                  href: '/docs/webfetch', 
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
            )
          ]
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
                    onClick: generateApiKey,
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
      
      // Login Modal
      showLoginModal && h(LoginModal, {
        onClose: () => setShowLoginModal(false),
        onLogin: () => {
          setShowLoginModal(false);
          window.location.reload(); // Recargar para verificar autenticación
        }
      }),
      
      // API Key Modal
      showKeyModal && h(ApiKeyModal, {
        apiKey: newApiKey,
        onClose: () => setShowKeyModal(false)
      })
    ].filter(Boolean)
  );
}

// Renderizar la aplicación
const appContainer = document.getElementById('app');
if (appContainer) {
  render(h(App), appContainer);
}
