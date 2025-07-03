package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"toolbox/auth"
	"toolbox/email"
)

var db *sql.DB

// SetupRoutes configura las rutas de la API en el enrutador proporcionado
func SetupRoutes(mux *http.ServeMux, database *sql.DB) {
	db = database

	// Configurar la base de datos en el paquete auth
	auth.SetDB(database)

	// Ruta para solicitar un enlace mágico
	mux.HandleFunc("/api/auth/request-magic-link", handleRequestMagicLink)

	// Ruta para validar un token mágico
	mux.HandleFunc("/api/auth/validate", handleValidateMagicLink)

	// Ruta para crear una nueva clave API
	mux.HandleFunc("/api/keys/create", handleCreateAPIKey)

	// Ruta para listar claves API
	mux.HandleFunc("/api/keys/list", handleListAPIKeys)

	// Ruta para revocar una clave API
	mux.HandleFunc("/api/keys/revoke/", handleRevokeAPIKey)

	// Ruta para obtener información del usuario autenticado
	mux.HandleFunc("/api/auth/me", handleGetCurrentUser)
}

// handleRequestMagicLink maneja la solicitud de un enlace mágico
func handleRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	// Configurar el encabezado de respuesta como JSON
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Método no permitido",
		})
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Solicitud inválida: " + err.Error(),
		})
		return
	}

	// Validar email
	if req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "El correo electrónico es requerido",
		})
		return
	}

	// Crear usuario si no existe
	if err := auth.CreateUser(req.Email); err != nil {
		errMsg := fmt.Sprintf("Error al crear usuario %s: %v", req.Email, err)
		log.Println(errMsg)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error al procesar la solicitud de inicio de sesión",
		})
		return
	}

	// Generar token mágico
	token, err := auth.GenerateRandomToken()
	if err != nil {
		log.Printf("Error al generar token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error al generar token de autenticación",
		})
		return
	}

	// Guardar token en la base de datos
	if err := auth.CreateMagicToken(req.Email, token); err != nil {
		log.Printf("Error al guardar token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error al procesar la solicitud de inicio de sesión",
		})
		return
	}

	// Enviar correo con el enlace mágico
	log.Printf("Intentando enviar correo a: %s", req.Email)
	err = email.SendMagicLink(req.Email, token, r.Host)
	if err != nil {
		log.Printf("Error al enviar correo: %v", err)

		// En desarrollo, devolver el enlace mágico directamente
		if os.Getenv("ENV") == "development" {
			magicLink := fmt.Sprintf("http://%s/api/auth/validate?token=%s", r.Host, token)
			json.NewEncoder(w).Encode(map[string]string{
				"message":    "Error al enviar correo (modo desarrollo)",
				"magic_link": magicLink,
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error al enviar el correo electrónico con el enlace de inicio de sesión",
		})
		return
	}

	// Responder al cliente con éxito
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Se ha enviado un enlace de inicio de sesión a tu correo electrónico",
	})
}

// handleValidateMagicLink valida un token mágico e inicia sesión
func handleValidateMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token no proporcionado", http.StatusBadRequest)
		return
	}

	// Validar token
	email, err := auth.ValidateMagicToken(token)
	if err != nil {
		http.Error(w, "Token inválido o expirado", http.StatusBadRequest)
		return
	}

	// Generar JWT
	jwtToken, err := auth.GenerateJWT(email)
	if err != nil {
		http.Error(w, "Error al generar token de sesión", http.StatusInternalServerError)
		return
	}

	// Configurar cookie de autenticación
	host := r.Host
	if host == "" {
		host = "localhost"
	}

	// Configurar la cookie con dominio y opciones de seguridad mejoradas
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    jwtToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		Secure:   false, // En producción, establece a true para HTTPS
		SameSite: http.SameSiteLaxMode,
	}

	// Solo configurar el dominio si no es localhost
	if !strings.Contains(host, "localhost") && !strings.HasPrefix(host, "127.0.0.1") {
		domain := strings.TrimPrefix(host, "www.")
		cookie.Domain = domain
	}

	http.SetCookie(w, cookie)

	// Crear una respuesta HTML que:
	// 1. Almacena el token en localStorage
	// 2. Redirige al dashboard
	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>Redirigiendo...</title>
		<script>
			document.addEventListener('DOMContentLoaded', function() {
				try {
					// Guardar el token en localStorage
					localStorage.setItem('token', '%s');
					
					// Verificar si la cookie se estableció correctamente
					const cookies = document.cookie.split(';').map(c => c.trim());
					const hasSessionCookie = cookies.some(c => c.startsWith('session_token='));
					
					if (!hasSessionCookie) {
						console.error('No se pudo establecer la cookie de sesión');
					}
					
					// Redirigir al dashboard
					window.location.href = '/dash';
				} catch (error) {
					console.error('Error al procesar la autenticación:', error);
					document.body.innerHTML = '<h1>Error</h1><p>Ocurrió un error al iniciar sesión. Por favor, inténtalo de nuevo.</p>';
				}
			});
		</script>
	</head>
	<body>
		<p>Iniciando sesión... Por favor, espera.</p>
	</body>
	</html>
	`, jwtToken)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
	return
}

// handleCreateAPIKey maneja la creación de una nueva clave API
func handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Solo permitir método POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obtener el email del usuario autenticado
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		// Si es una solicitud AJAX, devolver un error 401 con una URL de redirección
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "Se requiere autenticación",
				"redirect": "/login?redirect=" + url.QueryEscape(r.URL.Path),
			})
		} else {
			// Redirigir directamente para solicitudes normales
			http.Redirect(w, r, "/login?redirect="+url.QueryEscape(r.URL.Path), http.StatusFound)
		}
		return
	}

	// Leer el cuerpo de la solicitud
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	// Validar que se proporcione un nombre
	if req.Name == "" {
		http.Error(w, "Se requiere un nombre para la clave API", http.StatusBadRequest)
		return
	}

	// Obtener el ID del usuario
	userID, err := getUserIDByEmail(email)
	if err != nil {
		http.Error(w, "Error al obtener información del usuario", http.StatusInternalServerError)
		return
	}

	// Generar una nueva clave API
	key, err := auth.CreateAPIKey(userID, req.Name)
	if err != nil {
		http.Error(w, "Error al generar clave API: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Devolver la nueva clave API
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"api_key": key,
	})
}

// getAuthenticatedEmail obtiene el email del usuario autenticado ya sea por cookie o API key
func getAuthenticatedEmail(r *http.Request) (string, error) {
	// 1. Intentar obtener el token del encabezado Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Formato: "Bearer <token>"
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) == 2 && headerParts[0] == "Bearer" {
			token := headerParts[1]
			
			// Primero intentar validar como JWT
			claims, err := auth.ValidateToken(token)
			if err == nil {
				return claims.Email, nil
			}
			
			// Si no es un JWT válido, intentar como API key
			apiKey := token
			
			// Si la clave API tiene el formato ID.tbx_HASH, extraer el ID y el hash
			var keyID, keyHash string
			if strings.Contains(apiKey, ".tbx_") {
				keyParts := strings.Split(apiKey, ".tbx_")
				if len(keyParts) == 2 {
					keyID = keyParts[0]
					keyHash = "tbx_" + keyParts[1]
				}
			} else {
				// Si no tiene el formato esperado, asumir que es solo el ID
				keyID = apiKey
			}
			
			// Buscar el usuario por API key
			var email string
			err = db.QueryRow(
				`SELECT u.email 
				 FROM api_keys ak 
				 JOIN users u ON ak.user_id = u.id 
				 WHERE (ak.hash = ? OR ak.id = ?) 
				 AND ak.revoked = 0`,
				keyHash,
				keyID,
			).Scan(&email)

			if err != nil {
				if err == sql.ErrNoRows {
					return "", fmt.Errorf("API key inválida, expirada o revocada")
				}
				return "", fmt.Errorf("error al validar la API key: %v", err)
			}

			if email != "" {
				// Actualizar last_used_at
				_, _ = db.Exec(
					"UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?",
					keyID,
				)
				return email, nil
			}
		}
	}

	// 2. Intentar obtener el token de la cookie
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie != nil && cookie.Value != "" {
		// Validar el token JWT de la cookie
		claims, err := auth.ValidateToken(cookie.Value)
		if err == nil {
			return claims.Email, nil
		}
	}

	// 3. Intentar con token en el body (para peticiones JSON)
	if r.Header.Get("Content-Type") == "application/json" && r.ContentLength > 0 {
		var body struct {
			Token string `json:"token"`
		}

		// Guardar el body original para que pueda ser leído nuevamente
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&body); err == nil && body.Token != "" {
			claims, err := auth.ValidateToken(body.Token)
			if err == nil {
				return claims.Email, nil
			}
		}
	}

	// Si llegamos hasta aquí, no se pudo autenticar al usuario
	return "", fmt.Errorf("se requiere autenticación: %v", "token no válido o expirado")
}

// handleListAPIKeys maneja la lista de claves API del usuario
func handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Obtener el email del usuario autenticado
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		// Si es una solicitud AJAX, devolver un error 401 con una URL de redirección
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "Se requiere autenticación",
				"redirect": "/login?redirect=" + url.QueryEscape(r.URL.Path),
			})
		} else {
			// Redirigir directamente para solicitudes normales
			http.Redirect(w, r, "/login?redirect="+url.QueryEscape(r.URL.Path), http.StatusFound)
		}
		return
	}

	// Obtener el ID del usuario
	userID, err := getUserIDByEmail(email)
	if err != nil {
		http.Error(w, "Error al obtener información del usuario", http.StatusInternalServerError)
		return
	}

	// Obtener las claves API del usuario
	keys, err := auth.GetAPIKeys(userID)
	if err != nil {
		http.Error(w, "Error al obtener claves API: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Devolver la lista de claves API
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_keys": keys,
	})
}

// handleGetCurrentUser devuelve la información del usuario autenticado
func handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Obtener el email del usuario autenticado
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "No autorizado"})
		return
	}

	// Devolver la información del usuario
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"email": email,
	})
}

// handleRevokeAPIKey maneja la revocación de una clave API
func handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	// Configurar el encabezado de respuesta como JSON
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Método no permitido",
		})
		return
	}

	// Obtener el email del usuario autenticado
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Se requiere autenticación",
		})
		return
	}

	// Obtener el ID del usuario
	userID, err := getUserIDByEmail(email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error al obtener información del usuario",
		})
		return
	}

	// Obtener el ID de la clave a revocar del cuerpo de la petición
	var requestBody struct {
		KeyID string `json:"key_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Formato de solicitud inválido",
		})
		return
	}

	// Validar que se proporcionó un ID de clave
	if requestBody.KeyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Se requiere el ID de la clave a revocar",
		})
		return
	}

	// Revocar la clave
	err = auth.RevokeAPIKey(userID, requestBody.KeyID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error al revocar la clave: %v", err),
		})
		return
	}

	// Devolver éxito
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Clave API revocada exitosamente",
	})
}

// getUserIDByEmail busca el ID de usuario a partir del email
func getUserIDByEmail(email string) (int, error) {
	var userID int

	// Primero intentamos buscar el usuario
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err == nil {
		// Usuario encontrado
		return userID, nil
	}

	if err != sql.ErrNoRows {
		// Hubo un error en la consulta
		return 0, fmt.Errorf("error al buscar usuario: %v", err)
	}

	// Si llegamos aquí, el usuario no existe, así que lo creamos
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error al iniciar transacción: %v", err)
	}
	defer tx.Rollback()

	// Insertar el nuevo usuario
	result, err := tx.Exec("INSERT OR IGNORE INTO users (email) VALUES (?)", email)
	if err != nil {
		return 0, fmt.Errorf("error al crear usuario: %v", err)
	}

	// Obtener el ID del usuario recién creado
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error al obtener ID de usuario: %v", err)
	}

	// Si el ID es 0, significa que el usuario ya existía (por el OR IGNORE)
	if id == 0 {
		err = tx.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
		if err != nil {
			return 0, fmt.Errorf("error al obtener ID de usuario existente: %v", err)
		}
		return userID, tx.Commit()
	}

	// Todo salió bien, hacemos commit de la transacción
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error al hacer commit de la transacción: %v", err)
	}

	return int(id), nil
}

// respondJSON envía una respuesta JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
