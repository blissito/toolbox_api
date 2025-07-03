package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"toolbox/api"
	"toolbox/auth"
	"toolbox/database"

	"github.com/stretchr/testify/assert"
)

// getUserIDByEmail busca el ID de usuario a partir del email
func getUserIDByEmail(db *sql.DB, email string) (int, error) {
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

func TestWebFetchHTML(t *testing.T) {
	// Configurar una base de datos en memoria para los tests
	memoryDB, err := database.NewInMemoryDB()
	if err != nil {
		t.Fatal("Error al configurar la base de datos en memoria")
	}
	defer memoryDB.Close()

	// Ejecutar las migraciones
	err = database.RunMigrations(memoryDB.DB)
	if err != nil {
		t.Fatal("Error al ejecutar las migraciones:", err)
	}

	// Crear el manejador de enrutador
	mux := http.NewServeMux()
	api.SetupRoutes(mux, memoryDB.DB)

	// Crear un usuario de prueba y una API key
	userID, err := getUserIDByEmail(memoryDB.DB, "test@example.com")
	if err != nil {
		t.Fatal("Error al crear usuario de prueba:", err)
	}

	apiKey, err := auth.CreateAPIKey(userID, "test-key")
	if err != nil {
		t.Fatal("Error al crear API key:", err)
	}

	// Configuración del servidor de prueba para la herramienta webfetch
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html>
			<head>
				<title>Test Page</title>
				<meta name="description" content="Test description">
				<meta property="og:image" content="https://test.com/image.jpg">
			</head>
			<body>
				<h1>Mocked Content</h1>
			</body>
		</html>`))
	}))
	defer testServer.Close()

	// Crear el payload para la petición
	payload := map[string]interface{}{
		"tool": "webfetch",
		"payload": map[string]interface{}{
			"url":    testServer.URL,
			"format": "html",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err, "Error al serializar el payload")

	// Crear la petición HTTP
	req, err := http.NewRequest(
		"POST",
		"/api/tool",
		bytes.NewBuffer(payloadBytes),
	)
	assert.NoError(t, err, "Error al crear la petición")

	// Configurar headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "ToolboxTestSuite/1.0")

 // Crear el recorder para la respuesta
 rr := httptest.NewRecorder()

	// Ejecutar la petición
	mux.ServeHTTP(rr, req) // Usar el manejador configurado

	// Verificar el código de estado
	// Verificar código de estado directamente del mock
	assert.Equal(t, http.StatusOK, rr.Code, "Código de estado incorrecto")

	// Decodificar y validar respuesta
	var response struct {
		Output   string                 `json:"output"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "Error decodificando respuesta")
	
	// Verificar el código de estado HTTP
	assert.Equal(t, http.StatusOK, rr.Code, "Código de estado HTTP incorrecto")
	assert.NotEmpty(t, response.Output, "El output no debería estar vacío")

	// Verificar los metadatos
	// Verificar metadata usando la estructura
	metadata := response.Metadata
	assert.Equal(t, "Test Page", metadata["title"], "Título incorrecto")
	assert.Equal(t, "Test description", metadata["description"], "Descripción incorrecta")
	assert.Equal(t, "https://test.com/image.jpg", metadata["image"], "Imagen incorrecta")
	assert.Equal(t, "text/html", metadata["content_type"], "Content-Type incorrecto")
}
