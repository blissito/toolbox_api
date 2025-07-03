package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DB es la conexión a la base de datos
var (
	DB *sql.DB
)

// SetDB establece la conexión a la base de datos
func SetDB(database *sql.DB) {
	DB = database
}

// APIKey representa una clave API en el sistema
type APIKey struct {
	ID         string    `json:"id"`
	UserID     int       `json:"user_id"`
	Name       string    `json:"name"`
	Hash       string    `json:"-"` // No se expone en JSON
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
	Revoked    bool      `json:"revoked"`
}

// Configuración de autenticación
type Config struct {
	JWTSecret string
}

var config = Config{
	JWTSecret: getEnv("JWT_SECRET", "default-secret-key-change-me"),
}

// Claims para JWT
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// Inicializa la base de datos de autenticación
func InitDB(db *sql.DB) error {
	DB = db
	return createTables()
}

// Crea las tablas necesarias
func createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS magic_tokens (
			token TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (email) REFERENCES users(email) ON DELETE CASCADE
		)`,
		
		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP,
			revoked BOOLEAN DEFAULT FALSE,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
	}

	// Activar claves foráneas
	if _, err := DB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("error al activar claves foráneas: %v", err)
	}

	// Crear tablas
	for _, tableSQL := range tables {
		_, err := DB.Exec(tableSQL)
		if err != nil {
			return fmt.Errorf("error al crear tabla: %v\nSQL: %s", err, tableSQL)
		}
	}

	// Crear índices
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_magic_tokens_email ON magic_tokens(email)",
		"CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id)",
	}

	for _, indexSQL := range indexes {
		if _, err := DB.Exec(indexSQL); err != nil {
			return fmt.Errorf("error al crear índice: %v\nSQL: %s", err, indexSQL)
		}
	}

	return nil
}

// GenerateRandomToken genera un token aleatorio seguro
func GenerateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Genera un token JWT para el usuario
func GenerateJWT(email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.JWTSecret))
}

// Valida un token JWT
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token inválido")
	}

	return claims, nil
}

// Crea un nuevo usuario si no existe
func CreateUser(email string) error {
	// Verificar si el usuario ya existe
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if err != nil {
		return fmt.Errorf("error al verificar usuario existente: %v", err)
	}

	// Si el usuario ya existe, no hay necesidad de crearlo
	if count > 0 {
		return nil
	}

	// Insertar nuevo usuario
	_, err = DB.Exec("INSERT INTO users (email) VALUES (?)", email)
	if err != nil {
		return fmt.Errorf("error al crear usuario: %v", err)
	}

	return nil
}

// Crea un nuevo token mágico
func CreateMagicToken(email, token string) error {
	// Obtener el ID del usuario
	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		return fmt.Errorf("no se pudo encontrar el usuario con email %s: %v", email, err)
	}

	// Calcular la fecha de expiración (24 horas a partir de ahora)
	expiresAt := time.Now().Add(24 * time.Hour).UTC()

	// Eliminar tokens antiguos
	if _, err := DB.Exec("DELETE FROM magic_tokens WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("error al eliminar tokens antiguos: %v", err)
	}

	// Insertar nuevo token con fecha de expiración
	_, err = DB.Exec(
		"INSERT INTO magic_tokens (token, user_id, expires_at) VALUES (?, ?, ?)", 
		token, 
		userID, 
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("error al insertar nuevo token: %v", err)
	}

	return nil
}

// Valida un token mágico
func ValidateMagicToken(token string) (string, error) {
	// Verificar si el token está vacío
	if token == "" {
		return "", fmt.Errorf("token vacío")
	}

	// Primero, obtener el user_id del token
	var userID int
	var expiresAt time.Time

	// Consultar el token y verificar si ha expirado
	err := DB.QueryRow("SELECT user_id, expires_at FROM magic_tokens WHERE token = ? AND used = 0", token).
		Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("token no encontrado o ya ha sido usado")
	}
	if err != nil {
		log.Printf("Error al validar el token %s: %v", token, err)
		return "", fmt.Errorf("error al validar el token")
	}

	// Verificar si el token ha expirado
	if time.Now().After(expiresAt) {
		// Marcar el token como usado aunque haya expirado
		_, _ = DB.Exec("UPDATE magic_tokens SET used = 1 WHERE token = ?", token)
		return "", fmt.Errorf("el token ha expirado")
	}

	// Obtener el email del usuario
	var email string
	err = DB.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("usuario no encontrado")
		}
		log.Printf("Error al obtener el email del usuario: %v", err)
		return "", fmt.Errorf("error al validar el usuario")
	}

	// Marcar el token como usado
	if _, err := DB.Exec("UPDATE magic_tokens SET used = 1 WHERE token = ?", token); err != nil {
		log.Printf("Advertencia: no se pudo marcar el token como usado: %v", err)
	}

	return email, nil
}

// Crea una nueva clave API
func CreateAPIKey(userID int, name string) (string, error) {
	// Validar parámetros
	if userID <= 0 {
		return "", fmt.Errorf("se requiere un ID de usuario válido")
	}
	if name == "" {
		return "", fmt.Errorf("se requiere un nombre para la clave API")
	}

	// Generar un ID único para la clave API
	keyID, err := GenerateRandomToken()
	if err != nil {
		return "", fmt.Errorf("error al generar ID de clave: %v", err)
	}

	// Generar un token seguro
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("error al generar token: %v", err)
	}

	// Crear un hash del token
	hasher := sha256.New()
	hasher.Write(tokenBytes)
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Formato: tbx_<hash>
	key := fmt.Sprintf("tbx_%s", tokenHash[:32])

	// Almacenar en la base de datos
	_, err = DB.Exec(
		"INSERT INTO api_keys (id, user_id, name, hash) VALUES (?, ?, ?, ?)",
		keyID,
		userID,
		name,
		tokenHash,
	)

	if err != nil {
		return "", fmt.Errorf("error al guardar la clave API: %v", err)
	}

	// Devolver la clave en formato legible (solo se muestra una vez)
	return fmt.Sprintf("%s.%s", keyID, key), nil
}

// Obtiene las claves API de un usuario
func GetAPIKeys(userID int) ([]APIKey, error) {
	rows, err := DB.Query(`
		SELECT 
			id,
			user_id,
			name,
			created_at,
			last_used_at,
			revoked
		FROM api_keys 
		WHERE user_id = ? 
		ORDER BY created_at DESC`, 
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf("error al consultar claves API: %v", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		var lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Name,
			&key.CreatedAt,
			&lastUsedAt,
			&key.Revoked,
		)

		if err != nil {
			return nil, fmt.Errorf("error al escanear clave API: %v", err)
		}

		// Si lastUsedAt es válido, establecerlo en la estructura
		if lastUsedAt.Valid {
			key.LastUsedAt = lastUsedAt.Time
		}

		keys = append(keys, key)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error al iterar claves API: %v", err)
	}

	return keys, nil
}

// Obtiene una variable de entorno o un valor por defecto
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Responde con JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// RevokeAPIKey revoca una clave API
func RevokeAPIKey(userID int, keyID string) error {
    // Verificar que el usuario sea el propietario de la clave
    var dbUserID int
    err := DB.QueryRow(
        "SELECT user_id FROM api_keys WHERE id = ?",
        keyID,
    ).Scan(&dbUserID)

    if err != nil {
        if err == sql.ErrNoRows {
            return fmt.Errorf("clave API no encontrada")
        }
        return fmt.Errorf("error al verificar la clave API: %v", err)
    }

    if dbUserID != userID {
        return fmt.Errorf("no autorizado para revocar esta clave")
    }

    // Actualizar la clave como revocada
    _, err = DB.Exec(
        "UPDATE api_keys SET revoked = 1, last_used_at = CURRENT_TIMESTAMP WHERE id = ?",
        keyID,
    )

    if err != nil {
        return fmt.Errorf("error al revocar la clave API: %v", err)
    }

    return nil
}
