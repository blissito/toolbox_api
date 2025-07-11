package api

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
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
	"toolbox/tools"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/jaytaylor/html2text"
)

var db *sql.DB

// SetupRoutes configura las rutas de la API en el enrutador proporcionado
func SetupRoutes(mux *http.ServeMux, database *sql.DB) {
	db = database

	// Configurar la base de datos en el paquete auth
	auth.SetDB(database)

	// Authentication routes
	mux.HandleFunc("/api/auth/request-magic-link", handleRequestMagicLink)
	mux.HandleFunc("/api/auth/validate", handleValidateMagicLink)
	mux.HandleFunc("/api/auth/me", handleGetCurrentUser)
	mux.HandleFunc("/auth/logout", handleLogout)

	// API Key management routes
	mux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListAPIKeys(w, r)
		case http.MethodPost:
			handleCreateAPIKey(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// API Key deletion route (using DELETE method)
	mux.HandleFunc("/api/keys/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			handleDeleteAPIKey(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Ruta para herramientas como webfetch
	mux.HandleFunc("/api/tool", handleTool)
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

	// En desarrollo, mostrar el enlace mágico directamente
	if os.Getenv("ENV") == "development" {
		magicLink := fmt.Sprintf("http://%s/api/auth/validate?token=%s", r.Host, token)
		log.Printf("Enlace mágico (desarrollo): %s", magicLink)
		json.NewEncoder(w).Encode(map[string]string{
			"message":    "Enlace mágico (modo desarrollo)",
			"magic_link": magicLink,
		})
		return
	}

	// En producción, intentar enviar el correo
	log.Printf("Producción: Intentando enviar enlace mágico a %s", req.Email)

	err = email.SendMagicLink(req.Email, token, r.Host)
	if err != nil {
		log.Printf("Error al enviar correo en producción: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error al enviar el correo de inicio de sesión",
		})
		return
	}

	log.Printf("Correo de enlace mágico enviado exitosamente a %s", req.Email)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Se ha enviado un enlace de inicio de sesión al correo: " + req.Email,
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Iniciando sesión...</title>
	<script>
		document.addEventListener('DOMContentLoaded', function() {
			try {
				// Almacenar el token JWT en localStorage
				window.localStorage.setItem('jwt', '%s');
				
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
</html>`, jwtToken)
}

// generateAPIKey genera una nueva clave API en formato tbx_<random_chars>
func generateAPIKey() (string, error) {
	key, err := generateRandomString(16) // 16 caracteres aleatorios
	if err != nil {
		return "", fmt.Errorf("error generando clave API: %v", err)
	}
	return "tbx_" + key, nil
}

// generateRandomString genera una cadena aleatoria segura de la longitud especificada
func generateRandomString(length int) (string, error) {
	// Asegurarnos de tener suficientes bytes para la longitud solicitada
	// Cada byte se convierte a 1.33 caracteres en base64, así que necesitamos al menos length/1.33 bytes
	bytesNeeded := (length*6 + 7) / 8 // Cálculo seguro para asegurar suficientes caracteres
	if bytesNeeded < 1 {
		bytesNeeded = 1
	}

	b := make([]byte, bytesNeeded)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("error al leer bytes aleatorios: %v", err)
	}

	// Codificar a base64 y asegurarnos de que tenga al menos la longitud solicitada
	encoded := base64.URLEncoding.EncodeToString(b)
	if len(encoded) < length {
		// Si por alguna razón es más corto, concatenar con más caracteres aleatorios
		for len(encoded) < length {
			more, err := generateRandomString(length - len(encoded))
			if err != nil {
				return "", fmt.Errorf("error al generar cadena aleatoria adicional: %v", err)
			}
			encoded += more
		}
	}

	// Tomar exactamente la longitud solicitada
	return encoded[:length], nil
}

// getUserIDByEmail obtiene el ID de usuario por su email
func getUserIDByEmail(email string) (int64, error) {
	var userID int64
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("error al obtener ID de usuario: %v", err)
	}

	return userID, nil
}

// validateAPIKey valida una clave API y devuelve el email del usuario si es válida
func validateAPIKey(key string) (string, error) {
	// Verificar que la clave tenga el formato correcto
	if !strings.HasPrefix(key, "tbx_") {
		return "", fmt.Errorf("formato de clave API inválido")
	}

	var email string
	err := db.QueryRow(
		"SELECT u.email FROM users u "+
			"JOIN api_keys ak ON u.id = ak.user_id "+
			"WHERE ak.key = ? AND (ak.revoked = 0 OR ak.revoked IS NULL)",
		key,
	).Scan(&email)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("clave API no válida o revocada")
		}
		return "", fmt.Errorf("error validando la clave API: %v", err)
	}

	// Actualizar last_used_at
	_, err = db.Exec("UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE key = ?", key)
	if err != nil {
		log.Printf("Error actualizando last_used_at: %v", err)
		// Continuamos a pesar del error, no es crítico
	}

	return email, nil
}

// handleGetCurrentUser maneja la solicitud para obtener el usuario actual
func handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email, err := getAuthenticatedEmail(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"email": email,
	})
}

// handleCreateAPIKey maneja la creación de una nueva clave API
func handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Obtener el email del usuario autenticado
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Se requiere autenticación",
		})
		return
	}

	// Obtener el ID del usuario
	userID, err := getUserIDByEmail(email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Error al obtener el ID de usuario",
		})
		return
	}

	// Generar una nueva clave API
	apiKey, err := generateAPIKey()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Error al generar la clave API",
		})
		return
	}

	// Generar un ID único para la clave
	keyID, err := generateRandomString(8)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Error al generar el ID de la clave",
		})
		return
	}

	// Insertar la nueva clave en la base de datos
	_, err = db.Exec(
		"INSERT INTO api_keys (id, user_id, name, key, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
		keyID,
		userID,
		"Clave generada "+time.Now().Format("2006-01-02 15:04"),
		apiKey,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Error al guardar la clave API",
		})
		return
	}

	// Devolver la clave generada
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"key":     apiKey,
		"id":      keyID,
	})
}

// handleDeleteAPIKey maneja la eliminación de una clave API
func handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	// Obtener el ID de la clave de la URL
	keyID := strings.TrimPrefix(r.URL.Path, "/api/keys/")
	if keyID == "" {
		http.Error(w, "Se requiere el ID de la clave", http.StatusBadRequest)
		return
	}

	// Obtener el email del usuario autenticado
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	// Obtener el ID del usuario
	userID, err := getUserIDByEmail(email)
	if err != nil {
		http.Error(w, "Error al obtener el ID de usuario", http.StatusInternalServerError)
		return
	}

	// Verificar que la clave pertenece al usuario y eliminarla
	result, err := db.Exec(
		"DELETE FROM api_keys WHERE id = ? AND user_id = ?",
		keyID,
		userID,
	)

	if err != nil {
		http.Error(w, "Error al eliminar la clave API", http.StatusInternalServerError)
		return
	}

	// Verificar que se eliminó alguna fila
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error al verificar la eliminación", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Clave no encontrada o no tienes permiso para eliminarla", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}

// handleListAPIKeys maneja la lista de claves API del usuario
// handleLogout handles user logout by clearing the session cookie
func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete the cookie
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production", // Secure in production only
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to home page after logout
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Obtener el email del usuario autenticado
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Se requiere autenticación",
		})
		return
	}

	// Obtener el ID del usuario
	userID, err := getUserIDByEmail(email)
	if err != nil {
		http.Error(w, "Error al obtener el ID de usuario", http.StatusInternalServerError)
		return
	}

	// Obtener las claves del usuario desde la base de datos
	rows, err := db.Query(
		"SELECT id, name, key, created_at, last_used_at "+
			"FROM api_keys WHERE user_id = ? "+
			"ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		log.Printf("Error al consultar claves API: %v", err)
		http.Error(w, "Error al obtener las claves API", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Procesar los resultados
	var keys []map[string]interface{}
	for rows.Next() {
		var id, name, key string
		var createdAt, lastUsedAt sql.NullTime

		if err := rows.Scan(&id, &name, &key, &createdAt, &lastUsedAt); err != nil {
			log.Printf("Error escaneando fila: %v", err)
			continue
		}

		keyData := map[string]interface{}{
			"id":         id,
			"name":       name,
			"created_at": createdAt.Time.Format(time.RFC3339),
		}

		// Solo incluir la clave si fue generada recientemente (últimos 5 minutos)
		// y está en el almacenamiento local del navegador
		if time.Since(createdAt.Time) < 5*time.Minute {
			keyData["key"] = key
		}

		if lastUsedAt.Valid {
			keyData["last_used_at"] = lastUsedAt.Time.Format(time.RFC3339)
		}

		keys = append(keys, keyData)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterando sobre claves API: %v", err)
		http.Error(w, "Error al procesar las claves API", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"keys":    keys,
	})
}

func getAuthenticatedEmail(r *http.Request) (string, error) {
	// 1. Intentar obtener el token del encabezado Authorization
	authHeader := r.Header.Get("Authorization")
	// ... (rest of the code remains the same)
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
			email, err := validateAPIKey(token)
			if err == nil {
				return email, nil
			}

			// Si llegamos aquí, el token no es válido
			return "", fmt.Errorf("invalid or expired token")
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
		// Guardar el body original
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var body struct {
			Token string `json:"token"`
		}

		if err := json.Unmarshal(bodyBytes, &body); err == nil && body.Token != "" {
			// Primero intentar como JWT
			claims, err := auth.ValidateToken(body.Token)
			if err == nil {
				return claims.Email, nil
			}

			// Luego intentar como API key
			email, err := validateAPIKey(body.Token)
			if err == nil {
				return email, nil
			}
		}
	}

	// Si llegamos hasta aquí, no se pudo autenticar al usuario
	return "", fmt.Errorf("se requiere autenticación")
}

// handleTool maneja las solicitudes a /api/tool
func handleTool(w http.ResponseWriter, r *http.Request) {
	// Configurar el tipo de contenido de la respuesta
	w.Header().Set("Content-Type", "application/json")

	// Solo permitir método POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Método no permitido. Se requiere POST",
			"code":    "method_not_allowed",
		})
		return
	}

	// Verificar autenticación
	email, err := getAuthenticatedEmail(r)
	if err != nil {
		// Determinar el tipo de error de autenticación
		errMsg := "Se requiere autenticación"
		errCode := "unauthorized"
		statusCode := http.StatusUnauthorized

		// Si el token es inválido o ha expirado
		if err == sql.ErrNoRows || err.Error() == "token expirado" {
			errMsg = "Token inválido o expirado"
			errCode = "invalid_token"
		}

		// Si es una solicitud AJAX, devolver un error JSON
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  false,
			"error":    errMsg,
			"code":     errCode,
			"redirect": "/login?redirect=" + url.QueryEscape(r.URL.Path),
		})
		return
	}

	// Registrar el uso de la API para métricas
	_, _ = db.Exec("UPDATE api_keys SET last_used_at = datetime('now') WHERE user_id = (SELECT id FROM users WHERE email = ?)", email)

	// Decodificar el cuerpo de la solicitud
	var req struct {
		Tool    string                 `json:"tool"`
		Payload map[string]interface{} `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Error al decodificar el cuerpo de la solicitud",
			"code":    "invalid_request",
			"details": err.Error(),
		})
		return
	}

	// Verificar que se proporcionó una herramienta
	if req.Tool == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Se requiere el campo 'tool' en la solicitud",
			"code":    "missing_required_field",
			"field":   "tool",
		})
		return
	}

	// Manejar diferentes herramientas
	switch req.Tool {
	case "webfetch":
		handleWebFetch(w, req.Payload)
	case "screenshot":
		handleScreenshot(w, req.Payload)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Herramienta no soportada",
			"code":    "unsupported_tool",
			"tool":    req.Tool,
		})
	}
}

// handleScreenshot maneja la captura de pantalla de una URL
func handleScreenshot(w http.ResponseWriter, payload map[string]interface{}) {
	// Función para enviar errores estandarizados
	sendError := func(statusCode int, code, message string) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   message,
			"code":    code,
		})
	}

	// Obtener la URL del payload
	urlStr, ok := payload["url"].(string)
	if !ok || urlStr == "" {
		sendError(http.StatusBadRequest, "missing_url", "Se requiere el parámetro 'url' en el payload")
		return
	}

	// Validar que la URL sea válida
	_, err := url.ParseRequestURI(urlStr)
	if err != nil {
		sendError(http.StatusBadRequest, "invalid_url", "La URL proporcionada no es válida")
		return
	}

	// Tomar la captura de pantalla
	screenshot, err := tools.ShotScrapper(urlStr)
	if err != nil {
		sendError(http.StatusInternalServerError, "screenshot_failed", "Error al tomar la captura de pantalla: "+err.Error())
		return
	}

	// Establecer el tipo de contenido como imagen PNG
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	w.Write(screenshot)
}

// handleWebFetch maneja la herramienta webfetch
func handleWebFetch(w http.ResponseWriter, payload map[string]interface{}) {
	// Función para enviar errores estandarizados
	sendError := func(statusCode int, code, message string, details ...interface{}) {
		w.WriteHeader(statusCode)
		errResponse := map[string]interface{}{
			"success": false,
			"error":   message,
			"code":    code,
		}

		// Agregar detalles adicionales si se proporcionan
		if len(details) > 0 {
			errResponse["details"] = details[0]
		}

		json.NewEncoder(w).Encode(errResponse)
	}

	// Obtener la URL del payload
	urlStr, ok := payload["url"].(string)
	if !ok || urlStr == "" {
		sendError(http.StatusBadRequest, "missing_url", "Se requiere el parámetro 'url' en el payload", map[string]string{
			"field":   "url",
			"message": "El campo 'url' es requerido y no puede estar vacío",
		})
		return
	}

	// Validar la URL
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		sendError(http.StatusBadRequest, "invalid_url", "URL inválida. Debe comenzar con http:// o https://", map[string]string{
			"url":      urlStr,
			"expected": "URL debe comenzar con http:// o https://",
		})
		return
	}

	// Obtener el formato (opcional, por defecto "html")
	format := "html"
	if f, ok := payload["format"].(string); ok && f != "" {
		switch f {
		case "text", "markdown", "html":
			format = f
		default:
			sendError(http.StatusBadRequest, "invalid_format", "Formato no válido. Use 'text', 'markdown' o 'html'", map[string]interface{}{
				"format":   f,
				"accepted": []string{"text", "markdown", "html"},
			})
			return
		}
	}

	// Obtener el timeout (opcional, por defecto 30 segundos)
	timeout := 30
	if t, ok := payload["timeout"].(float64); ok && t > 0 {
		timeout = int(t)
		if timeout > 120 { // Máximo 2 minutos
			timeout = 120
		}
	}

	// Configurar el cliente HTTP con timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Crear la solicitud
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		sendError(http.StatusInternalServerError, "request_creation_failed", "Error al crear la solicitud HTTP", map[string]string{
			"details": err.Error(),
		})
		return
	}

	// Añadir headers para parecer un navegador
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.8,en-US;q=0.5,en;q=0.3")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	// Realizar la petición
	resp, err := client.Do(req)
	if err != nil {
		sendError(http.StatusBadGateway, "request_failed", "No se pudo completar la solicitud al servidor remoto", map[string]string{
			"url":     urlStr,
			"details": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(http.StatusInternalServerError, "read_response_failed", "Error al leer la respuesta del servidor remoto", map[string]string{
			"details": err.Error(),
		})
		return
	}

	// Determinar el tipo de contenido
	contentType := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml+xml")

	// Convertir el contenido según el formato solicitado
	var result string
	var conversionError error

	switch format {
	case "html":
		result = string(body)
	case "markdown":
		if isHTML {
			converter := md.NewConverter("", true, nil)
			result, conversionError = converter.ConvertString(string(body))
		} else {
			result = string(body)
		}
	case "text":
		if isHTML {
			result, conversionError = html2text.FromString(string(body), html2text.Options{
				PrettyTables: true,
				TextOnly:     true,
			})
		} else {
			result = string(body)
		}
	}

	// Manejar errores de conversión
	if conversionError != nil {
		// Si hay un error en la conversión, devolver el contenido original
		result = string(body)
	}

	// Extraer metadatos de la página
	metadata := map[string]interface{}{
		"url":            urlStr,
		"format":         format,
		"content_type":   contentType,
		"status_code":    resp.StatusCode,
		"content_length": len(body),
	}

	// Si es HTML, extraer metadatos adicionales
	if isHTML {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err == nil {
			// Extraer título
			if title := doc.Find("title").First().Text(); title != "" {
				metadata["title"] = strings.TrimSpace(title)
			}

			// Extraer imagen de Open Graph (og:image)
			if ogImage, exists := doc.Find("meta[property='og:image']").First().Attr("content"); exists && ogImage != "" {
				metadata["image"] = ogImage
			} else if twitterImage, exists := doc.Find("meta[name='twitter:image']").First().Attr("content"); exists && twitterImage != "" {
				// Si no hay og:image, intentar con twitter:image
				metadata["image"] = twitterImage
			} else if icon := doc.Find("link[rel*='icon']").First(); icon != nil {
				// Si no hay imágenes en meta tags, usar el favicon
				if iconHref, exists := icon.Attr("href"); exists && iconHref != "" {
					// Convertir a URL absoluta si es relativa
					if base, err := url.Parse(urlStr); err == nil {
						if iconURL, err := base.Parse(iconHref); err == nil {
							metadata["image"] = iconURL.String()
						}
					}
				}
			}

			// Extraer descripción si está disponible
			if desc, exists := doc.Find("meta[property='og:description']").First().Attr("content"); exists && desc != "" {
				metadata["description"] = strings.TrimSpace(desc)
			} else if desc, exists := doc.Find("meta[name='description']").First().Attr("content"); exists && desc != "" {
				metadata["description"] = strings.TrimSpace(desc)
			}
		}
	}

	// Crear la respuesta exitosa
	response := map[string]interface{}{
		"success":  true,
		"output":   result,
		"metadata": metadata,
	}

	// Si hubo un error de conversión, incluirlo como advertencia
	if conversionError != nil {
		response["warning"] = map[string]string{
			"code":    "conversion_warning",
			"message": "Se produjo un error al convertir el contenido",
			"details": conversionError.Error(),
		}
	}

	// Enviar la respuesta como JSON
	json.NewEncoder(w).Encode(response)
}

// ...
