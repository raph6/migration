package migration

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
)

const (
	MySQL      = "mysql"
	PostgreSQL = "postgres"
	SQLite     = "sqlite3"
	PGX        = "pgx"
)

func Migrate(db *sqlx.DB) error {
	dn := db.DriverName()
	if dn != MySQL && dn != PostgreSQL && dn != SQLite && dn != PGX {
		return fmt.Errorf("migration: driver %s not supported", dn)
	}

	if err := createMigrationsTable(db, dn); err != nil {
		return err
	}

	// get all files in migrations folder
	files, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}

	// loop through all files
	for _, file := range files {
		// get the filename
		filename := file.Name()

		// if not sql file, skip it
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		// get all before the first underscore
		idMigration := filename[:strings.Index(filename, "_")]

		// check if the migration has already been executed
		// if the migration has already been executed, skip it
		if isImported(db, idMigration, dn) {
			continue
		}

		// read the file
		content, err := os.ReadFile(fmt.Sprintf("migrations/%s", filename))
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")

		var statement string
		inDollarBlock := false
		dollarTag := ""
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			if strings.HasPrefix(trimmed, "--") {
				continue // skip comments
			}

			if !inDollarBlock && strings.Contains(trimmed, "$$") {
				idx := strings.Index(trimmed, "$")
				endIdx := strings.Index(trimmed[idx+1:], "$")
				if endIdx != -1 {
					dollarTag = trimmed[idx : idx+endIdx+2]
					inDollarBlock = true
					statement += line + "\n"
					continue
				}
			}

			if inDollarBlock {
				statement += line + "\n"

				if strings.Contains(trimmed, dollarTag) {
					stmt := strings.TrimSpace(statement)
					if stmt != "" {
						_, err := db.Exec(stmt)
						if err != nil {
							return err
						}
						fmt.Println(filename + ": sql executed")
					}

					statement = ""
					inDollarBlock = false
					dollarTag = ""
				}
				continue
			}

			statement += line + " "
			if strings.HasSuffix(trimmed, ";") {
				stmt := strings.TrimSpace(statement)
				if stmt != "" {
					_, err := db.Exec(stmt)
					if err != nil {
						return err
					}
					fmt.Println(filename + ": sql executed")
				}
				statement = ""
			}
		}

		// add the migration to the migrations table
		if err := insertMigration(db, idMigration, dn); err != nil {
			return err
		}
	}

	fmt.Println("migrations done")

	return nil
}

func createMigrationsTable(db *sqlx.DB, dn string) error {
	var createTableQuery string

	switch dn {
	case MySQL:
		createTableQuery = `
			CREATE TABLE IF NOT EXISTS migrations (
				id INT AUTO_INCREMENT PRIMARY KEY,
				id_migration VARCHAR(255) NOT NULL,
				executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	case PostgreSQL, PGX:
		createTableQuery = `
			CREATE TABLE IF NOT EXISTS migrations (
				id SERIAL PRIMARY KEY,
				id_migration VARCHAR(255) NOT NULL,
				executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	case SQLite:
		createTableQuery = `
			CREATE TABLE IF NOT EXISTS migrations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				id_migration TEXT NOT NULL,
				executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	}

	_, err := db.Exec(createTableQuery)
	if err != nil {
		return err
	}
	fmt.Println("table `migrations` initialized")
	return nil
}

func insertMigration(db *sqlx.DB, idMigration, dn string) error {
	var query string
	switch dn {
	case MySQL:
		query = "INSERT INTO migrations (id_migration, executed_at) VALUES (?, NOW())"
	case PostgreSQL, PGX:
		query = "INSERT INTO migrations (id_migration, executed_at) VALUES ($1, NOW())"
	case SQLite:
		query = "INSERT INTO migrations (id_migration, executed_at) VALUES (?, datetime('now'))"
	}

	_, err := db.Exec(query, idMigration)
	return err
}

func isImported(db *sqlx.DB, idMigration, dn string) bool {
	var count int

	switch dn {
	case MySQL:
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = ?", idMigration)
	case PostgreSQL, PGX:
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = $1", idMigration)
	case SQLite:
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = ?", idMigration)
	}
	return count > 0
}
