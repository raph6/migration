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
)

func Migrate(db *sqlx.DB) {
	dn := db.DriverName()
	if dn != MySQL && dn != PostgreSQL && dn != SQLite {
		panic(fmt.Sprintf("migration: driver %s not supported\n", dn))
	}

	createMigrationsTable(db, dn)

	// get all files in migrations folder
	files, err := os.ReadDir("migrations")
	if err != nil {
		panic(err)
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
		content, err := os.ReadFile("migrations/" + filename)
		if err != nil {
			panic(err)
		}

		// execute the migration
		sqlStmtSlice := strings.Split(string(content), ";")
		for _, request := range sqlStmtSlice {
			request := strings.TrimSpace(request)
			if request != "" {
				db.MustExec(request)
				fmt.Println(filename + ": sql executed")
			}
		}

		// add the migration to the migrations table
		insertMigration(db, idMigration, dn)
	}

	fmt.Println("migrations done")
}

func createMigrationsTable(db *sqlx.DB, dn string) {
	var createTableQuery string

	switch dn {
	case MySQL:
		createTableQuery = `
			CREATE TABLE IF NOT EXISTS migrations (
				id INT AUTO_INCREMENT PRIMARY KEY,
				id_migration VARCHAR(255) NOT NULL,
				executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	case PostgreSQL:
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

	db.MustExec(createTableQuery)

	fmt.Println("table `migrations` inited")
}

func insertMigration(db *sqlx.DB, idMigration, dn string) {
	var query string
	switch dn {
	case MySQL:
		query = "INSERT INTO migrations (id_migration, executed_at) VALUES (?, NOW())"
	case PostgreSQL:
		query = "INSERT INTO migrations (id_migration, executed_at) VALUES ($1, NOW())"
	case SQLite:
		query = "INSERT INTO migrations (id_migration, executed_at) VALUES (?, datetime('now'))"
	}

	db.MustExec(query, idMigration)
}

func isImported(db *sqlx.DB, idMigration, dn string) bool {
	var count int

	switch dn {
	case MySQL:
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = ?", idMigration)
	case PostgreSQL:
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = $1", idMigration)
	case SQLite:
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = ?", idMigration)
	}
	return count > 0
}
