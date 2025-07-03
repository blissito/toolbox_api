package database

import (
	"database/sql"
	"fmt"
)

// RunMigrations ejecuta todas las migraciones necesarias en la base de datos
func RunMigrations(db *sql.DB) error {
	// Crear tabla de migraciones si no existe
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("error al crear tabla de migraciones: %v", err)
	}

	// Obtener la versión actual de la base de datos
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("error al obtener versión actual de la base de datos: %v", err)
	}

	// Ejecutar migraciones pendientes
	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
				CREATE TABLE IF NOT EXISTS users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					email TEXT UNIQUE NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS magic_tokens (
					token TEXT PRIMARY KEY,
					user_id INTEGER NOT NULL,
					expires_at TIMESTAMP NOT NULL,
					used BOOLEAN DEFAULT FALSE,
					FOREIGN KEY (user_id) REFERENCES users(id)
				);

				CREATE TABLE IF NOT EXISTS api_keys (
					id TEXT PRIMARY KEY,
					user_id INTEGER NOT NULL,
					name TEXT NOT NULL,
					hash TEXT NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					last_used_at TIMESTAMP,
					revoked BOOLEAN DEFAULT FALSE,
					FOREIGN KEY (user_id) REFERENCES users(id)
				);

				CREATE INDEX IF NOT EXISTS idx_magic_tokens_token ON magic_tokens(token);
				CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
			`,
		},
		// Agregar más migraciones aquí según sea necesario
	}

	// Ejecutar migraciones pendientes
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error al iniciar transacción: %v", err)
	}
	defer tx.Rollback()

	for _, migration := range migrations {
		if migration.version > currentVersion {
			// Ejecutar migración
			if _, err := tx.Exec(migration.sql); err != nil {
				return fmt.Errorf("error al ejecutar migración %d: %v", migration.version, err)
			}

			// Registrar migración
			if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.version); err != nil {
				return fmt.Errorf("error al registrar migración %d: %v", migration.version, err)
			}
		}
	}

	return tx.Commit()
}
