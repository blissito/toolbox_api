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
					key TEXT NOT NULL UNIQUE,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					last_used_at TIMESTAMP,
					revoked BOOLEAN DEFAULT FALSE,
					FOREIGN KEY (user_id) REFERENCES users(id)
				);

				CREATE INDEX IF NOT EXISTS idx_magic_tokens_token ON magic_tokens(token);
				CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
			`,
		},
		{
			version: 2,
			sql: `
			PRAGMA foreign_keys=off;

			-- 1. Create a backup of the existing data
			CREATE TABLE IF NOT EXISTS api_keys_backup AS SELECT * FROM api_keys;

			-- 2. Create new table structure with the key column
			CREATE TABLE IF NOT EXISTS new_api_keys (
				id TEXT PRIMARY KEY,
				user_id INTEGER NOT NULL,
				name TEXT NOT NULL,
				key TEXT NOT NULL UNIQUE,  -- This replaces the hash column
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				last_used_at TIMESTAMP,
				revoked BOOLEAN DEFAULT FALSE,
				FOREIGN KEY (user_id) REFERENCES users(id)
			);

			-- 3. Copy data from backup to new table
			-- For existing records, use the id as the key since hash column doesn't exist
			INSERT INTO new_api_keys (id, user_id, name, key, created_at, last_used_at, revoked)
			SELECT 
				id, 
				user_id, 
				name, 
				id as key,  -- Use id as the key since hash doesn't exist
				created_at, 
				last_used_at, 
				revoked 
			FROM api_keys_backup;

			-- 4. Drop old table and rename new one
			DROP TABLE IF EXISTS api_keys;
			ALTER TABLE new_api_keys RENAME TO api_keys;

			-- 5. Recreate indexes
			CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
			CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key);

			-- 6. Clean up
			DROP TABLE IF EXISTS api_keys_backup;
			PRAGMA foreign_keys=on;
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
