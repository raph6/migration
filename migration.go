package migration

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
)

func Migrate(db *sqlx.DB) {
	dn := db.DriverName()
	if dn != "mysql" && dn != "postgres" && dn != "sqlite3" {
		panic(fmt.Sprintf("migration: driver %s not supported\n", dn))
	}

	initTable(db)

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
		if isImported(db, idMigration) {
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
		insertMigration(db, idMigration)
	}

	fmt.Println("migrations done")
}

func initTable(db *sqlx.DB) {
	dn := db.DriverName()

	if dn == "mysql" {
		db.MustExec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			id_migration VARCHAR(255) NOT NULL,
			executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	} else if dn == "postgres" {
		db.MustExec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			id_migration VARCHAR(255) NOT NULL,
			executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	} else if dn == "sqlite3" {
		db.MustExec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			id_migration TEXT NOT NULL,
			executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	}

	fmt.Println("table `migrations` inited")
}

func insertMigration(db *sqlx.DB, idMigration string) {
	dn := db.DriverName()
	if dn == "mysql" {
		db.MustExec("INSERT INTO migrations (id_migration, executed_at) VALUES (?, NOW())", idMigration)
	} else if dn == "postgres" {
		db.MustExec("INSERT INTO migrations (id_migration, executed_at) VALUES ($1, NOW())", idMigration)
	} else if dn == "sqlite3" {
		db.MustExec("INSERT INTO migrations (id_migration, executed_at) VALUES (?, datetime('now'))", idMigration)
	}
}

func isImported(db *sqlx.DB, idMigration string) bool {
	var count int
	dn := db.DriverName()
	if dn == "mysql" {
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = ?", idMigration)
	} else if dn == "postgres" {
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = $1", idMigration)
	} else if dn == "sqlite3" {
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = ?", idMigration)
	}
	return count > 0
}
