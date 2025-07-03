package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Init inicializa la conexión a la base de datos SQLite
func Init(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error al abrir la base de datos: %v", err)
	}

	// Verificar la conexión
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error al hacer ping a la base de datos: %v", err)
	}

	// Configurar la base de datos para mejor rendimiento
	if _, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		return nil, fmt.Errorf("error al configurar la base de datos: %v", err)
	}

	return db, nil
}

// Close cierra la conexión a la base de datos de manera segura
func Close(db *sql.DB) error {
	if db != nil {
		// Cerrar la base de datos de manera limpia
		if _, err := db.Exec(`PRAGMA optimize`); err != nil {
			return fmt.Errorf("error al optimizar la base de datos: %v", err)
		}
		return db.Close()
	}
	return nil
}

// NewInMemoryDB crea una nueva base de datos en memoria SQLite para tests
func NewInMemoryDB() (*DBWrapper, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("error al crear base de datos en memoria: %v", err)
	}

	// Configurar la base de datos para mejor rendimiento
	if _, err := db.Exec(`
		PRAGMA journal_mode = MEMORY;
		PRAGMA synchronous = OFF;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		return nil, fmt.Errorf("error al configurar la base de datos en memoria: %v", err)
	}

	// Verificar la conexión
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error al hacer ping a la base de datos en memoria: %v", err)
	}

	return &DBWrapper{DB: db}, nil
}

// DBWrapper es un wrapper para la base de datos
type DBWrapper struct {
	*sql.DB
}
