package migration

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
)

func Migrate(db *sqlx.DB) {
	db.MustExec(`
	CREATE TABLE IF NOT EXISTS migrations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		id_migration VARCHAR(255) NOT NULL,
		executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	fmt.Println("table `migrations` inited")

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
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM migrations WHERE id_migration = ?", idMigration)

		// if the migration has already been executed, skip it
		if count > 0 {
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
		db.MustExec("INSERT INTO migrations (id_migration, executed_at) VALUES (?, NOW())", idMigration)
	}

	fmt.Println("migrations done")
}
